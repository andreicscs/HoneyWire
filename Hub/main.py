"""
HoneyWire Hub — main.py
Universal sensor hub supporting arbitrary sensor types via the HoneyWire Event Standard.

Architecture notes:
  - DB schema is versioned via PRAGMA user_version; add new migrations to MIGRATIONS list.
  - Notifications are dispatched through notify(); add new backends to NOTIFICATION_BACKENDS.
  - Auth is centralised in verify_ui_auth(); cookie logic lives in one place.
  - The Event model is intentionally sensor-agnostic: source, target, details cover every
    conceivable sensor (network tarpit, file canary, LLM probe, email honeypot, …).
"""

import secrets
import sqlite3
import os
import json
import hashlib
import logging
from contextlib import contextmanager
from datetime import datetime, timedelta, timezone
from typing import Any, Callable

import requests
from fastapi import Depends, FastAPI, Header, HTTPException, Request, BackgroundTasks
from fastapi.responses import HTMLResponse, JSONResponse
from fastapi.templating import Jinja2Templates
from pydantic import BaseModel, field_validator

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger("honeywire")

# ---------------------------------------------------------------------------
# Configuration  (all values come from environment variables)
# ---------------------------------------------------------------------------
API_SECRET        = os.getenv("API_SECRET", "super_secret_key_123")
DASHBOARD_PASSWORD = os.getenv("DASHBOARD_PASSWORD", "")
NTFY_URL          = os.getenv("NTFY_URL", "")
GOTIFY_URL        = os.getenv("GOTIFY_URL", "")
GOTIFY_TOKEN      = os.getenv("GOTIFY_TOKEN", "")
DB_PATH           = os.getenv("DB_PATH", "/data/honeywire.db")

# Versioning: one source of truth stored in root file + env override.
DEFAULT_VERSION   = "1.0.0"
VERSION_FILE_PATH = os.path.join(os.path.dirname(__file__), "..", "VERSION")
try:
    with open(os.path.abspath(VERSION_FILE_PATH), "r") as f:
        FILE_VERSION = f.read().strip()
except FileNotFoundError:
    FILE_VERSION = DEFAULT_VERSION

HONEYWIRE_VERSION = os.getenv("HONEYWIRE_VERSION", FILE_VERSION)

AUTH_COOKIE_NAME  = "hw_auth"

# Store active sessions in memory to prevent Pass-the-Hash vulnerabilities
# Format: { "session_token": expiration_datetime }
ACTIVE_SESSIONS: dict[str, datetime] = {}

# ---------------------------------------------------------------------------
# Database — versioned migrations
# ---------------------------------------------------------------------------
MIGRATIONS: list[str] = [
    # v1 — initial schema
    """
    CREATE TABLE IF NOT EXISTS events (
        id           INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp    TEXT    NOT NULL,
        sensor_id    TEXT    NOT NULL,
        sensor_type  TEXT    NOT NULL DEFAULT 'generic',
        event_type   TEXT    NOT NULL DEFAULT 'alert',
        severity     TEXT    NOT NULL DEFAULT 'medium',
        source       TEXT    NOT NULL DEFAULT 'Unknown',
        target       TEXT    NOT NULL DEFAULT 'Unknown',
        action_taken TEXT    NOT NULL DEFAULT 'logged',
        details      TEXT    NOT NULL DEFAULT '{}',
        is_read      INTEGER NOT NULL DEFAULT 0
    );
    CREATE TABLE IF NOT EXISTS sensors (
        sensor_id   TEXT PRIMARY KEY,
        last_seen   TEXT NOT NULL,
        sensor_type TEXT NOT NULL DEFAULT 'generic',
        metadata    TEXT NOT NULL DEFAULT '{}'
    );
    CREATE TABLE IF NOT EXISTS config (
        key   TEXT PRIMARY KEY,
        value TEXT NOT NULL
    );
    INSERT OR IGNORE INTO config (key, value) VALUES ('is_armed', 'true');
    """
]


def _get_db_version(conn: sqlite3.Connection) -> int:
    return conn.execute("PRAGMA user_version").fetchone()[0]


def _set_db_version(conn: sqlite3.Connection, version: int) -> None:
    conn.execute(f"PRAGMA user_version = {version}")


def run_migrations() -> None:
    """Apply any pending schema migrations on startup."""
    conn = sqlite3.connect(DB_PATH)
    try:
        current = _get_db_version(conn)
        pending  = MIGRATIONS[current:]
        if not pending:
            log.info("DB schema is up to date (version %d).", current)
            return
        for i, migration in enumerate(pending, start=current + 1):
            log.info("Applying DB migration to version %d …", i)
            conn.executescript(migration)
            _set_db_version(conn, i)
        conn.commit()
        log.info("DB migrations complete. Schema now at version %d.", current + len(pending))
    finally:
        conn.close()


@contextmanager
def get_db():
    """Context manager that yields a sqlite3 connection and closes it on exit."""
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    try:
        yield conn
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()


run_migrations()

# ---------------------------------------------------------------------------
# Notification dispatcher
# ---------------------------------------------------------------------------
def _notify_ntfy(title: str, message: str, severity: str) -> None:
    if not NTFY_URL:
        return
    
    # Ntfy scale: 1 (min) to 5 (max)
    priorities = {"info": 1, "low": 2, "medium": 3, "high": 4, "critical": 5}
    priority = priorities.get(severity.lower(), 3) # Default to medium
    
    headers = {"Title": title, "Priority": str(priority), "Tags": "rotating_light"}
    
    NTFY_TOKEN = os.getenv("NTFY_TOKEN", "")
    if NTFY_TOKEN:
        headers["Authorization"] = f"Bearer {NTFY_TOKEN}"

    try:
        response = requests.post(NTFY_URL, data=message.encode('utf-8'), headers=headers, timeout=5)
        response.raise_for_status() 
        
    except requests.exceptions.HTTPError as exc:
        log.warning("Ntfy rejected the request: %s | Details: %s", exc, response.text)
    except Exception as exc:
        log.warning("Ntfy failed to connect entirely: %s", exc)


def _notify_gotify(title: str, message: str, severity: str) -> None:
    if not (GOTIFY_URL and GOTIFY_TOKEN):
        return
        
    # Gotify scale: 0-3 (min/silent), 4-7 (default/sound), 8-10 (high/interrupt)
    priorities = {"info": 1, "low": 3, "medium": 5, "high": 8, "critical": 10}
    priority = priorities.get(severity.lower(), 5) # Default to medium

    try:
        response = requests.post(
            GOTIFY_URL,
            headers={"X-Gotify-App-Token": GOTIFY_TOKEN},
            json={"title": title, "message": message, "priority": priority},
            timeout=5,
        )
        response.raise_for_status()
        
    except requests.exceptions.HTTPError as exc:
        log.warning("Gotify rejected the request: %s | Details: %s", exc, response.text)
    except Exception as exc:
        log.warning("Gotify failed to connect entirely: %s", exc)


NOTIFICATION_BACKENDS: list[Callable[[str, str, str], None]] = [
    _notify_ntfy,
    _notify_gotify,
]


def notify(title: str, message: str, severity: str) -> None:
    """Dispatch a notification to all configured backends. 
       NOTE: This should always be called via BackgroundTasks to prevent blocking."""
    for backend in NOTIFICATION_BACKENDS:
        backend(title, message, severity)


# ---------------------------------------------------------------------------
# Pydantic models
# ---------------------------------------------------------------------------
class Event(BaseModel):
    contract_version: str
    sensor_id:   str
    sensor_type: str = "generic"
    event_type:  str = "alert"
    severity:    str = "medium"
    source:      str = "Unknown"
    target:      str = "Unknown"
    action_taken: str = "logged"
    details:     dict = {}

    @field_validator("severity")
    @classmethod
    def validate_severity(cls, v: str) -> str:
        allowed = {"info", "low", "medium", "high", "critical"}
        if v.lower() not in allowed:
            raise ValueError(f"severity must be one of {allowed}")
        return v.lower()


class Heartbeat(BaseModel):
    sensor_id:   str
    sensor_type: str = "generic"
    metadata:    Any = {}


class SystemState(BaseModel):
    is_armed: bool


class LoginRequest(BaseModel):
    password: str


# ---------------------------------------------------------------------------
# FastAPI app
# ---------------------------------------------------------------------------
app = FastAPI(title="HoneyWire Hub")
templates = Jinja2Templates(directory="templates")


# ---------------------------------------------------------------------------
# Auth — Centralized Security Logic
# ---------------------------------------------------------------------------
def verify_ui_auth(request: Request) -> None:
    """Validates the presence of an active, unexpired session token."""
    if DASHBOARD_PASSWORD:
        cookie_val = request.cookies.get(AUTH_COOKIE_NAME)
        if not cookie_val or cookie_val not in ACTIVE_SESSIONS:
            raise HTTPException(status_code=401, detail="Unauthorized")
        if datetime.now() > ACTIVE_SESSIONS[cookie_val]:
            del ACTIVE_SESSIONS[cookie_val]
            raise HTTPException(status_code=401, detail="Session Expired")


def verify_agent_auth(x_api_key: str = Header(None)) -> None:
    """Validates sensor API keys using constant-time comparison."""
    if not x_api_key or not secrets.compare_digest(x_api_key, API_SECRET):
        raise HTTPException(status_code=401, detail="Invalid API key")


# ---------------------------------------------------------------------------
# Frontend API
# ---------------------------------------------------------------------------
@app.get("/api/v1/system/state", dependencies=[Depends(verify_ui_auth)])
async def get_system_state():
    with get_db() as conn:
        row = conn.execute("SELECT value FROM config WHERE key='is_armed'").fetchone()
    return {"is_armed": row["value"] == "true"}


@app.patch("/api/v1/system/state", dependencies=[Depends(verify_ui_auth)])
async def set_system_state(state: SystemState):
    with get_db() as conn:
        conn.execute(
            "UPDATE config SET value=? WHERE key='is_armed'",
            ("true" if state.is_armed else "false",),
        )
    return {"status": "success", "is_armed": state.is_armed}


@app.get("/api/v1/version", dependencies=[Depends(verify_ui_auth)])
async def get_version():
    return {"version": HONEYWIRE_VERSION}


@app.get("/api/v1/sensors", dependencies=[Depends(verify_ui_auth)])
async def get_sensors():
    with get_db() as conn:
        rows = conn.execute(
            "SELECT sensor_id, sensor_type, last_seen, metadata FROM sensors ORDER BY sensor_id"
        ).fetchall()

    now = datetime.now()
    fleet = []
    for r in rows:
        last_seen_dt = datetime.strptime(r["last_seen"], "%Y-%m-%d %H:%M:%S")
        fleet.append({
            "sensor_id":   r["sensor_id"],
            "sensor_type": r["sensor_type"],
            "last_seen":   r["last_seen"],
            "metadata":    json.loads(r["metadata"]),
            "status":      "online" if (now - last_seen_dt) < timedelta(seconds=90) else "offline",
        })
    return fleet


@app.get("/api/v1/events", dependencies=[Depends(verify_ui_auth)])
async def get_events():
    with get_db() as conn:
        rows = conn.execute("SELECT * FROM events ORDER BY id DESC").fetchall()
    return [
        {
            "contract_version": r["contract_version"],
            "id":          r["id"],
            "timestamp":   r["timestamp"],
            "sensor_id":   r["sensor_id"],
            "sensor_type": r["sensor_type"],
            "event_type":  r["event_type"],
            "severity":    r["severity"],
            "source":      r["source"],
            "target":      r["target"],
            "action_taken": r["action_taken"],
            "details":     json.loads(r["details"]) if r["details"] else {},
            "is_read":     bool(r["is_read"]),
        }
        for r in rows
    ]


@app.patch("/api/v1/events/read", dependencies=[Depends(verify_ui_auth)])
async def mark_events_read():
    with get_db() as conn:
        conn.execute("UPDATE events SET is_read = 1 WHERE is_read = 0")
    return {"status": "success"}


@app.patch("/api/v1/events/{event_id}/read", dependencies=[Depends(verify_ui_auth)])
async def mark_single_event_read(event_id: int):
    with get_db() as conn:
        conn.execute("UPDATE events SET is_read = 1 WHERE id = ?", (event_id,))
    return {"status": "success"}


@app.delete("/api/v1/events", dependencies=[Depends(verify_ui_auth)])
async def clear_events():
    with get_db() as conn:
        conn.execute("DELETE FROM events")
    return {"status": "success"}


# ---------------------------------------------------------------------------
# Sensor API
# ---------------------------------------------------------------------------
@app.post("/api/v1/heartbeat", dependencies=[Depends(verify_agent_auth)])
async def receive_heartbeat(hb: Heartbeat):
    now = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    metadata_json = json.dumps(hb.metadata) if isinstance(hb.metadata, (dict, list)) else "{}"
    with get_db() as conn:
        conn.execute(
            """INSERT INTO sensors (sensor_id, sensor_type, last_seen, metadata)
               VALUES (?, ?, ?, ?)
               ON CONFLICT(sensor_id) DO UPDATE SET last_seen=?, sensor_type=?, metadata=?""",
            (hb.sensor_id, hb.sensor_type, now, metadata_json,
             now, hb.sensor_type, metadata_json),
        )
    return {"status": "alive"}


@app.post("/api/v1/event", dependencies=[Depends(verify_agent_auth)])
async def receive_event(event: Event, bg_tasks: BackgroundTasks):
    timestamp    = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%S")
    details_json = json.dumps(event.details) if isinstance(event.details, (dict, list)) else str(event.details)

    with get_db() as conn:
        conn.execute(
            """INSERT INTO events
               (timestamp, sensor_id, sensor_type, event_type, severity,
                source, target, action_taken, details, is_read)
               VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)""",
            (timestamp, event.sensor_id, event.sensor_type, event.event_type,
             event.severity, event.source, event.target, event.action_taken, details_json),
        )
        is_armed = conn.execute(
            "SELECT value FROM config WHERE key='is_armed'"
        ).fetchone()["value"] == "true"

    msg = (
        f"[{event.sensor_id}] {event.event_type.upper()} — "
        f"{event.source} → {event.target} | action: {event.action_taken}"
    )
    
    if is_armed:
        bg_tasks.add_task(notify, title=f"HoneyWire Alert ({event.severity.upper()})", message=msg, severity=event.severity)

    log.info("Event received: %s", msg if is_armed else f"{event.sensor_id}/{event.event_type}")
    return {"status": "success"}


# ---------------------------------------------------------------------------
# Web UI & Vault Door
# ---------------------------------------------------------------------------
LOGIN_HTML = """<!DOCTYPE html>
<html lang="en" class="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>HoneyWire Sentinel | Authentication</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;700&family=Plus+Jakarta+Sans:wght@400;600;800&display=swap" rel="stylesheet">
    <script>
        tailwind.config = { darkMode: 'class' }
        if (localStorage.getItem('theme') === 'light' || (!('theme' in localStorage) && !window.matchMedia('(prefers-color-scheme: dark)').matches)) {
            document.documentElement.classList.remove('dark')
        } else {
            document.documentElement.classList.add('dark')
        }
    </script>
    <style>
        body { font-family: 'Plus Jakarta Sans', sans-serif; }
        .mono { font-family: 'JetBrains Mono', monospace; }
        .bg-grid {
            background-size: 40px 40px;
            background-image: linear-gradient(to right, rgba(161, 161, 170, 0.05) 1px, transparent 1px),
                              linear-gradient(to bottom, rgba(161, 161, 170, 0.05) 1px, transparent 1px);
        }
        .dark .bg-grid {
            background-image: linear-gradient(to right, rgba(255, 255, 255, 0.02) 1px, transparent 1px),
                              linear-gradient(to bottom, rgba(255, 255, 255, 0.02) 1px, transparent 1px);
        }
    </style>
</head>
<body class="bg-zinc-50 dark:bg-[#050507] text-zinc-900 dark:text-zinc-300 h-screen flex items-center justify-center bg-grid transition-colors duration-300">
    <button onclick="toggleTheme()" class="absolute top-6 right-6 p-2 rounded-xl bg-white dark:bg-zinc-900/50 border border-zinc-200 dark:border-zinc-800 text-zinc-500 hover:text-zinc-900 dark:hover:text-white shadow-sm transition-all backdrop-blur-md">
        <svg id="moon-icon" class="w-5 h-5 hidden dark:block" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path></svg>
        <svg id="sun-icon" class="w-5 h-5 block dark:hidden" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path></svg>
    </button>
    <div class="relative w-full max-w-md p-8 sm:p-12 bg-white/80 dark:bg-zinc-900/40 border border-zinc-200 dark:border-zinc-800/50 rounded-[2.5rem] shadow-2xl backdrop-blur-xl">
        <div class="text-center mb-10">
            <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-zinc-100 dark:bg-emerald-500/10 border border-zinc-200 dark:border-emerald-500/20 text-3xl mb-6 shadow-inner">🕸️</div>
            <h1 class="text-2xl font-black text-zinc-900 dark:text-white tracking-tighter">HONEYWIRE</h1>
            <p class="text-[10px] font-bold text-zinc-500 uppercase tracking-[0.25em] mt-2">Sentinel</p>
        </div>
        <form onsubmit="doLogin(event)" class="flex flex-col gap-5">
            <div class="space-y-2">
                <label for="pwd" class="text-[10px] font-black text-zinc-600 dark:text-zinc-500 uppercase tracking-widest pl-1">Authentication Key</label>
                <input type="password" id="pwd" placeholder="Enter passphrase..."
                       class="w-full p-4 rounded-xl bg-zinc-100 dark:bg-zinc-950/50 border border-zinc-300 dark:border-zinc-800 text-sm mono text-zinc-900 dark:text-zinc-200 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent transition-all placeholder-zinc-400 dark:placeholder-zinc-700" required>
            </div>
            <button type="submit"
                    class="w-full mt-2 py-4 rounded-xl bg-zinc-900 dark:bg-emerald-500/10 text-white dark:text-emerald-500 border border-transparent dark:border-emerald-500/20 hover:bg-zinc-800 dark:hover:bg-emerald-500/20 text-[10px] font-black uppercase tracking-[0.25em] transition-all shadow-lg dark:shadow-[0_0_15px_rgba(16,185,129,0.1)]">
                Login
            </button>
        </form>
        <div id="err" class="hidden mt-6 p-4 rounded-xl bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 text-center">
            <p class="text-[10px] font-black text-rose-600 dark:text-rose-500 uppercase tracking-widest">Access Denied: Invalid Key</p>
        </div>
    </div>
    <script>
        function toggleTheme() {
            if (document.documentElement.classList.contains('dark')) {
                document.documentElement.classList.remove('dark');
                localStorage.setItem('theme', 'light');
            } else {
                document.documentElement.classList.add('dark');
                localStorage.setItem('theme', 'dark');
            }
        }
        async function doLogin(e) {
            e.preventDefault();
            const btn = e.target.querySelector('button[type="submit"]');
            const originalText = btn.innerText;
            btn.innerText = "AUTHENTICATING...";
            btn.disabled = true;
            btn.classList.add('opacity-75');
            try {
                const res = await fetch('/login', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({password: document.getElementById('pwd').value})
                });
                if (res.ok) {
                    btn.innerText = "ACCESS GRANTED";
                    btn.classList.replace('dark:text-emerald-500', 'dark:text-zinc-950');
                    btn.classList.replace('dark:bg-emerald-500/10', 'dark:bg-emerald-500');
                    btn.classList.replace('bg-zinc-900', 'bg-emerald-500');
                    setTimeout(() => window.location.reload(), 400);
                } else {
                    document.getElementById('err').classList.remove('hidden');
                    btn.innerText = originalText;
                    btn.disabled = false;
                    btn.classList.remove('opacity-75');
                    document.getElementById('pwd').value = '';
                    document.getElementById('pwd').focus();
                }
            } catch (err) {
                btn.innerText = originalText;
                btn.disabled = false;
            }
        }
    </script>
</body>
</html>"""


@app.post("/login")
async def login(req: LoginRequest):
    if DASHBOARD_PASSWORD and secrets.compare_digest(req.password, DASHBOARD_PASSWORD):
        session_token = secrets.token_hex(32)
        ACTIVE_SESSIONS[session_token] = datetime.now() + timedelta(days=30)
        
        response = JSONResponse(content={"status": "ok"})
        response.set_cookie(
            key=AUTH_COOKIE_NAME, value=session_token,
            max_age=2_592_000, httponly=True, samesite="strict",
        )
        return response
    raise HTTPException(status_code=401, detail="Invalid Password")


@app.get("/logout")
async def logout():
    response = HTMLResponse(content="<script>window.location.href='/';</script>")
    response.delete_cookie(AUTH_COOKIE_NAME)
    return response


@app.get("/", response_class=HTMLResponse)
async def serve_dashboard(request: Request):
    if DASHBOARD_PASSWORD:
        cookie_val = request.cookies.get(AUTH_COOKIE_NAME)
        if not cookie_val or cookie_val not in ACTIVE_SESSIONS or datetime.now() > ACTIVE_SESSIONS[cookie_val]:
            return HTMLResponse(content=LOGIN_HTML)
    return templates.TemplateResponse(request=request, name="index.html")


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)