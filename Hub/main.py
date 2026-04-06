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
from datetime import datetime, timedelta, timezone
from fastapi import FastAPI, Request, HTTPException, Depends, Header, BackgroundTasks, Query
from fastapi import Query
from datetime import timedelta

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger("honeywire")

# ---------------------------------------------------------------------------
# Configuration  (all values come from HW_ environment variables)
# ---------------------------------------------------------------------------
# The API Key sensors use to authenticate to the Hub
API_SECRET         = os.getenv("HW_HUB_KEY", "change_this_to_a_secure_random_string")

# Dashboard UI Password
DASHBOARD_PASSWORD = os.getenv("HW_DASHBOARD_PASSWORD", "admin")

# Notification Endpoints
NTFY_URL           = os.getenv("HW_NTFY_URL", "")
GOTIFY_URL         = os.getenv("HW_GOTIFY_URL", "")
GOTIFY_TOKEN       = os.getenv("HW_GOTIFY_TOKEN", "")

# Database Location
DB_PATH            = os.getenv("HW_DB_PATH", "/data/honeywire.db")

# Versioning: one source of truth stored in root file + env override.
DEFAULT_VERSION   = "1.0.0"
VERSION_FILE_PATH = os.path.join(os.path.dirname(__file__), "..", "VERSION")
try:
    with open(os.path.abspath(VERSION_FILE_PATH), "r") as f:
        FILE_VERSION = f.read().strip()
except FileNotFoundError:
    FILE_VERSION = DEFAULT_VERSION

# Use HW_VERSION to allow users to override it if strictly necessary
HONEYWIRE_VERSION = os.getenv("HW_VERSION", FILE_VERSION)

AUTH_COOKIE_NAME  = "hw_auth"

# Store active sessions in memory to prevent Pass-the-Hash vulnerabilities
# Format: { "session_token": expiration_datetime }
ACTIVE_SESSIONS: dict[str, datetime] = {}
# ---------------------------------------------------------------------------
# Database — versioned migrations
# ---------------------------------------------------------------------------
# We use one clean initialization string for new deployments
INIT_SCHEMA = """
CREATE TABLE IF NOT EXISTS events (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp        TEXT    NOT NULL,
    contract_version TEXT    NOT NULL DEFAULT '1.0.0',
    sensor_id        TEXT    NOT NULL,
    sensor_type      TEXT    NOT NULL DEFAULT 'generic',
    event_type       TEXT    NOT NULL DEFAULT 'alert',
    severity         TEXT    NOT NULL DEFAULT 'medium',
    source           TEXT    NOT NULL DEFAULT 'Unknown',
    target           TEXT    NOT NULL DEFAULT 'Unknown',
    action_taken     TEXT    NOT NULL DEFAULT 'logged',
    details          TEXT    NOT NULL DEFAULT '{}',
    is_read          INTEGER NOT NULL DEFAULT 0,
    is_archived INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sensors (
    sensor_id   TEXT PRIMARY KEY,
    first_seen  TEXT,
    last_seen   TEXT NOT NULL,
    sensor_type TEXT NOT NULL DEFAULT 'generic',
    metadata    TEXT NOT NULL DEFAULT '{}',
    is_silenced INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT OR IGNORE INTO config (key, value) VALUES ('is_armed', 'true');

CREATE TABLE IF NOT EXISTS sensor_heartbeats (
    sensor_id   TEXT NOT NULL,
    time_bucket TEXT NOT NULL,
    PRIMARY KEY (sensor_id, time_bucket)
);
CREATE INDEX IF NOT EXISTS idx_heartbeats_time ON sensor_heartbeats(time_bucket);
"""

def _get_db_version(conn: sqlite3.Connection) -> int:
    return conn.execute("PRAGMA user_version").fetchone()[0]

def _set_db_version(conn: sqlite3.Connection, version: int) -> None:
    conn.execute(f"PRAGMA user_version = {version}")

def init_db():
    """Run exactly once on startup to ensure tables exist."""
    conn = sqlite3.connect(DB_PATH)
    try:
        conn.executescript(INIT_SCHEMA)
        conn.commit()
        log.info("Database initialized successfully.")
    except Exception as e:
        log.error(f"Database initialization failed: {e}")
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

init_db()

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
    metadata:    dict = {}

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


def verify_agent_auth(x_api_key: str = Header(None), authorization: str = Header(None)):
    """Verifies the secret key sent by the sensor (Supports X-Api-Key and Bearer)."""
    token = x_api_key
    if not token and authorization and authorization.startswith("Bearer "):
        token = authorization.split(" ", 1)[1].strip()

    if not token or not secrets.compare_digest(token, API_SECRET):
        log.warning("Unauthorized access attempt detected.")
        raise HTTPException(status_code=401, detail="Unauthorized")

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


@app.get("/api/v1/version")
async def get_version():
    return {"version": HONEYWIRE_VERSION}


@app.get("/api/v1/sensors", dependencies=[Depends(verify_ui_auth)])
async def get_sensors():
    with get_db() as conn:
        # FIX 1: Add is_silenced to the SQL SELECT statement
        rows = conn.execute(
            "SELECT sensor_id, sensor_type, last_seen, metadata, is_silenced FROM sensors ORDER BY sensor_id"
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
            # FIX 2: Pass the boolean value to the frontend
            "is_silenced": bool(r["is_silenced"])
        })
    return fleet


@app.get("/api/v1/events", dependencies=[Depends(verify_ui_auth)])
async def get_events(archived: bool = Query(False)):
    with get_db() as conn:
        rows = conn.execute(
            "SELECT * FROM events WHERE is_archived = ? ORDER BY id DESC", 
            (1 if archived else 0,)
        ).fetchall()
        
    return [
        {
            "contract_version": r["contract_version"],
            "id":           r["id"],
            "timestamp":    r["timestamp"],
            "sensor_id":    r["sensor_id"],
            "sensor_type":  r["sensor_type"],
            "event_type":   r["event_type"],
            "severity":     r["severity"],
            "source":       r["source"],
            "target":       r["target"],
            "action_taken": r["action_taken"],
            "details":      json.loads(r["details"]) if r["details"] else {},
            "is_read":      bool(r["is_read"]),
            "is_archived":  bool(r["is_archived"]),
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

@app.get("/api/v1/uptime", dependencies=[Depends(verify_ui_auth)])
async def get_uptime_data(timeframe: str = Query("24H")):
    now = datetime.now(timezone.utc)
    
    if timeframe == "1H":
        num_blocks = 60
        delta = timedelta(minutes=1)
        fmt = "%Y-%m-%d %H:%M" 
        expected_pings = 1
    elif timeframe == "24H":
        num_blocks = 24
        delta = timedelta(hours=1)
        fmt = "%Y-%m-%d %H"    
        expected_pings = 60
    elif timeframe == "7D":
        num_blocks = 7
        delta = timedelta(days=1)
        fmt = "%Y-%m-%d"       
        expected_pings = 1440
    else: # 30D
        num_blocks = 30
        delta = timedelta(days=1)
        fmt = "%Y-%m-%d"       
        expected_pings = 1440

    cutoff = now - (delta * num_blocks)
    cutoff_str = cutoff.strftime("%Y-%m-%d %H:%M:%S")

    with get_db() as conn:
        sensors = conn.execute("SELECT sensor_id, last_seen, first_seen FROM sensors ORDER BY sensor_id").fetchall()
        rows = conn.execute(
            "SELECT sensor_id, time_bucket FROM sensor_heartbeats WHERE time_bucket >= ?",
            (cutoff_str,)
        ).fetchall()

    history = {s["sensor_id"]: {} for s in sensors}
    for r in rows:
        bucket_dt = datetime.strptime(r["time_bucket"], "%Y-%m-%d %H:%M:00")
        time_key = bucket_dt.strftime(fmt)
        history[r["sensor_id"]][time_key] = history[r["sensor_id"]].get(time_key, 0) + 1

    result = []
    for s in sensors:
        sensor_id = s["sensor_id"]
        blocks = []
        
        # Safely parse first_seen (fallback to 'now' if missing)
        first_seen_str = s["first_seen"] or now.strftime("%Y-%m-%d %H:%M:%S")
        first_seen_dt = datetime.strptime(first_seen_str, "%Y-%m-%d %H:%M:%S").replace(tzinfo=timezone.utc)
        
        # CRITICAL FIX: Convert first_seen to the exact same bucket format as time_key
        first_seen_key = first_seen_dt.strftime(fmt)
        
        for i in range(num_blocks - 1, -1, -1):
            block_time = now - (delta * i)
            time_key = block_time.strftime(fmt)
            
            time_label = "Current" if i == 0 else f"{i} " + ("mins ago" if timeframe == "1H" else "hours ago" if timeframe == "24H" else "days ago")
            
            # 1. NO DATA CHECK: Elegant string comparison
            if time_key < first_seen_key:
                status, label = "nodata", "No Data (Not Deployed Yet)"
            else:
                # 2. DEGRADED / UP / DOWN CHECK
                ping_count = history[sensor_id].get(time_key, 0)
                
                if ping_count == 0:
                    status, label = "down", "Offline"
                elif ping_count < (expected_pings * 0.85):
                    status, label = "degraded", f"Degraded ({ping_count}/{expected_pings} pings)"
                else:
                    status, label = "up", "Online"

            blocks.append({
                "status": status,
                "timeLabel": time_label,
                "label": label
            })

        # Override the very last block to strictly match live last_seen status
        last_seen_dt = datetime.strptime(s["last_seen"], "%Y-%m-%d %H:%M:%S").replace(tzinfo=timezone.utc)
        is_live = (now - last_seen_dt) < timedelta(seconds=90)
        blocks[-1] = {
            "status": "up" if is_live else "down",
            "timeLabel": "Current",
            "label": "Online (Live)" if is_live else "Offline (Live)"
        }

        result.append({
            "id": sensor_id,
            "name": sensor_id,
            "isOnline": is_live,
            "blocks": blocks
        })

    return result

@app.patch("/api/v1/events/{event_id}/archive", dependencies=[Depends(verify_ui_auth)])
async def archive_single_event(event_id: int):
    with get_db() as conn:
        conn.execute("UPDATE events SET is_archived = 1, is_read = 1 WHERE id = ?", (event_id,))
    return {"status": "success"}

@app.patch("/api/v1/events/archive-all", dependencies=[Depends(verify_ui_auth)])
async def archive_all_events():
    with get_db() as conn:
        # Only archive currently active events
        conn.execute("UPDATE events SET is_archived = 1, is_read = 1 WHERE is_archived = 0")
    return {"status": "success"}



class SilenceRequest(BaseModel):
    is_silenced: bool

@app.patch("/api/v1/sensors/{sensor_id}/silence", dependencies=[Depends(verify_ui_auth)])
async def toggle_sensor_silence(sensor_id: str, req: SilenceRequest):
    with get_db() as conn:
        conn.execute(
            "UPDATE sensors SET is_silenced = ? WHERE sensor_id = ?",
            (1 if req.is_silenced else 0, sensor_id)
        )
    return {"status": "success", "sensor_id": sensor_id, "is_silenced": req.is_silenced}

# ---------------------------------------------------------------------------
# Sensor API
# ---------------------------------------------------------------------------
@app.post("/api/v1/heartbeat", dependencies=[Depends(verify_agent_auth)])
async def receive_heartbeat(hb: Heartbeat):
    now = datetime.now(timezone.utc)
    now_str = now.strftime("%Y-%m-%d %H:%M:%S")
    
    # Round down to the current minute (e.g., "2026-04-05 21:05:00")
    minute_bucket = now.strftime("%Y-%m-%d %H:%M:00") 

    metadata_json = json.dumps(hb.metadata) if isinstance(hb.metadata, (dict, list)) else "{}"
    
    with get_db() as conn:
        # 1. Update current live status & explicitly set first_seen if new
        conn.execute(
            """INSERT INTO sensors (sensor_id, first_seen, last_seen, sensor_type, metadata)
               VALUES (?, ?, ?, ?, ?)
               ON CONFLICT(sensor_id) DO UPDATE SET last_seen=?, sensor_type=?, metadata=?""",
            (hb.sensor_id, now_str, now_str, hb.sensor_type, metadata_json,
             now_str, hb.sensor_type, metadata_json),
        )
        
        # 2. Log historical bucket (Ignores duplicates within the same minute)
        conn.execute(
            "INSERT OR IGNORE INTO sensor_heartbeats (sensor_id, time_bucket) VALUES (?, ?)",
            (hb.sensor_id, minute_bucket)
        )
        
    return {"status": "alive"}


@app.post("/api/v1/event", dependencies=[Depends(verify_agent_auth)])
async def receive_event(event: Event, bg_tasks: BackgroundTasks):
    # Optional but recommended: Semantic versioning check
    hub_major_version = HONEYWIRE_VERSION.split('.')[0]
    agent_major_version = event.contract_version.split('.')[0]
    
    if hub_major_version != agent_major_version:
        log.warning(f"Version mismatch rejected: Hub is {HONEYWIRE_VERSION}, Agent sent {event.contract_version}")
        raise HTTPException(
            status_code=426, 
            detail=f"Upgrade Required: Hub is on v{HONEYWIRE_VERSION}, but agent uses v{event.contract_version}"
        )

    timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M:%S")
    details_payload = event.metadata if hasattr(event, 'metadata') else {}
    details_json = json.dumps(details_payload) if isinstance(details_payload, (dict, list)) else str(details_payload)

    with get_db() as conn:
        conn.execute(
            """INSERT INTO events
               (timestamp, contract_version, sensor_id, sensor_type, event_type, severity,
                source, target, action_taken, details, is_read)
               VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)""",
            (timestamp, event.contract_version, event.sensor_id, event.sensor_type, event.event_type,
             event.severity, event.source, event.target, event.action_taken, details_json),
        )
        # 1. Check Global Arm State
        is_armed = conn.execute(
            "SELECT value FROM config WHERE key='is_armed'"
        ).fetchone()["value"] == "true"
        
        # 2. Check Specific Sensor Silence State
        sensor_row = conn.execute(
            "SELECT is_silenced FROM sensors WHERE sensor_id = ?", 
            (event.sensor_id,)
        ).fetchone()
        is_silenced = bool(sensor_row["is_silenced"]) if sensor_row else False

    msg = (
        f"[{event.sensor_id}] {event.event_type.upper()} — "
        f"{event.source} → {event.target} | action: {event.action_taken}"
    )
    
    if is_armed and not is_silenced:
        bg_tasks.add_task(notify, title=f"HoneyWire Alert ({event.severity.upper()})", message=msg, severity=event.severity)

    log.info("Event received (v%s): %s", event.contract_version, msg if is_armed else f"{event.sensor_id}/{event.event_type}")
    return {"status": "success"}


# ---------------------------------------------------------------------------
# Web UI & Vault Door (Uncodixified)
# ---------------------------------------------------------------------------
LOGIN_HTML = """<!DOCTYPE html>
<html lang="en" class="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>HoneyWire Sentinel | Authentication</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
    <script>
        tailwind.config = { darkMode: 'class' }
        if (localStorage.getItem('theme') === 'light' || (!('theme' in localStorage) && !window.matchMedia('(prefers-color-scheme: dark)').matches)) {
            document.documentElement.classList.remove('dark')
        } else {
            document.documentElement.classList.add('dark')
        }
    </script>
    <style>
        body { font-family: 'Inter', sans-serif; }
        .mono { font-family: 'JetBrains Mono', monospace; font-weight: 500; }
        .bg-grid {
            background-size: 40px 40px;
            background-image: linear-gradient(to right, rgba(148, 163, 184, 0.08) 1px, transparent 1px),
                              linear-gradient(to bottom, rgba(148, 163, 184, 0.08) 1px, transparent 1px);
        }
        .dark .bg-grid {
            background-image: linear-gradient(to right, rgba(255, 255, 255, 0.02) 1px, transparent 1px),
                              linear-gradient(to bottom, rgba(255, 255, 255, 0.02) 1px, transparent 1px);
        }
    </style>
</head>
<body class="bg-slate-100 dark:bg-[#0a0a0c] text-slate-700 dark:text-zinc-200 h-screen flex flex-col items-center justify-center bg-grid transition-colors duration-200 p-6">
    
    <div class="w-full max-w-[400px] space-y-8">
        <div class="flex flex-col items-center">
            <div class="w-12 h-12 flex items-center justify-center rounded-lg bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 shadow-sm text-2xl mb-4">🕸️</div>
            <h1 class="text-xl font-bold text-slate-900 dark:text-white tracking-tight">HoneyWire Sentinel</h1>
            <p class="text-sm text-slate-500 dark:text-zinc-500 mt-1">Authorized personnel only</p>
        </div>

        <div class="bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 rounded-lg shadow-sm p-8 transition-colors duration-200">
            <form onsubmit="doLogin(event)" class="space-y-6">
                <div class="space-y-2">
                    <label for="pwd" class="text-xs font-semibold text-slate-700 dark:text-zinc-400">Authentication Key</label>
                    <input type="password" id="pwd" placeholder="••••••••••••"
                           class="w-full px-3 py-2 rounded-md bg-slate-50 dark:bg-zinc-950 border border-slate-300 dark:border-zinc-800 text-sm mono text-slate-900 dark:text-zinc-200 focus:outline-none focus:ring-2 focus:ring-slate-400 dark:focus:ring-zinc-600 focus:border-transparent transition-all placeholder-slate-300 dark:placeholder-zinc-700" required>
                </div>
                
                <button type="submit"
                        class="w-full py-2 rounded-md bg-slate-900 dark:bg-zinc-100 text-white dark:text-zinc-900 hover:bg-slate-800 dark:hover:bg-white text-sm font-semibold transition-all shadow-sm">
                    Sign in
                </button>
            </form>

            <div id="err" class="hidden mt-6 p-3 rounded-md bg-rose-50 dark:bg-rose-900/20 border border-rose-200 dark:border-rose-800/30 text-center">
                <p class="text-xs font-medium text-rose-700 dark:text-rose-400">Access Denied: Invalid Key</p>
            </div>
        </div>

        <div class="flex justify-center">
            <button onclick="toggleTheme()" class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-slate-200 dark:bg-zinc-800 border border-slate-300 dark:border-zinc-700 text-slate-600 dark:text-zinc-400 hover:text-slate-900 dark:hover:text-white text-xs font-medium transition-colors">
                <svg id="moon-icon" class="w-3.5 h-3.5 hidden dark:block" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path></svg>
                <svg id="sun-icon" class="w-3.5 h-3.5 block dark:hidden" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path></svg>
                <span>Switch Theme</span>
            </button>
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
            btn.innerText = "Authenticating...";
            btn.disabled = true;
            btn.classList.add('opacity-50');
            
            try {
                const res = await fetch('/login', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({password: document.getElementById('pwd').value})
                });
                if (res.ok) {
                    btn.innerText = "Access Granted";
                    btn.classList.replace('bg-slate-900', 'bg-emerald-600');
                    btn.classList.replace('dark:bg-zinc-100', 'dark:bg-emerald-600');
                    btn.classList.add('text-white');
                    setTimeout(() => window.location.reload(), 400);
                } else {
                    document.getElementById('err').classList.remove('hidden');
                    btn.innerText = originalText;
                    btn.disabled = false;
                    btn.classList.remove('opacity-50');
                    document.getElementById('pwd').value = '';
                    document.getElementById('pwd').focus();
                }
            } catch (err) {
                btn.innerText = originalText;
                btn.disabled = false;
                btn.classList.remove('opacity-50');
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