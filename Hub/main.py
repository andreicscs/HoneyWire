import sqlite3
import os
import requests
import json
import hashlib
from fastapi import FastAPI, Header, HTTPException, Request, Depends
from fastapi.responses import HTMLResponse, JSONResponse
from fastapi.templating import Jinja2Templates
from pydantic import BaseModel
from typing import Any
from datetime import datetime, timedelta

# --- Configuration ---
API_SECRET = os.getenv("API_SECRET", "super_secret_key_123")
NTFY_URL = os.getenv("NTFY_URL", "")
GOTIFY_URL = os.getenv("GOTIFY_URL", "")
GOTIFY_TOKEN = os.getenv("GOTIFY_TOKEN", "")

# --- Vault Door Config ---
DASHBOARD_PASSWORD = os.getenv("DASHBOARD_PASSWORD", "")
AUTH_COOKIE_NAME = "hw_auth"
# Hash the password in memory so we aren't storing raw passwords in cookies
EXPECTED_HASH = hashlib.sha256(DASHBOARD_PASSWORD.encode()).hexdigest() if DASHBOARD_PASSWORD else ""

app = FastAPI(title="HoneyWire Hub")
templates = Jinja2Templates(directory="templates")

# --- Database Setup ---
def init_db():
    conn = sqlite3.connect("/data/nanotrap.db")
    c = conn.cursor()
    c.execute('''CREATE TABLE IF NOT EXISTS events
                 (id INTEGER PRIMARY KEY AUTOINCREMENT,
                  timestamp TEXT, sensor_id TEXT, sensor_ip TEXT, attacker_ip TEXT, 
                  target_port INTEGER, protocol TEXT, action TEXT, 
                  duration_sec REAL, raw_payload TEXT, is_read INTEGER DEFAULT 0)''')
    c.execute('''CREATE TABLE IF NOT EXISTS sensors
                 (sensor_id TEXT PRIMARY KEY, sensor_ip TEXT, last_seen TEXT)''')
    c.execute('''CREATE TABLE IF NOT EXISTS config (key TEXT PRIMARY KEY, value TEXT)''')
    c.execute("INSERT OR IGNORE INTO config (key, value) VALUES ('is_armed', 'true')")
    conn.commit()
    conn.close()

init_db()

# --- Data Model ---
class Event(BaseModel):
    sensor_id: str
    sensor_ip: str = "Unknown"
    attacker_ip: str
    target_port: int
    protocol: str = "TCP"
    action: str = "dropped"
    duration_sec: float = 0.0
    raw_payload: Any = ""

class Heartbeat(BaseModel):
    sensor_id: str
    sensor_ip: str

class SystemState(BaseModel):
    is_armed: bool

class LoginRequest(BaseModel):
    password: str

# --- Auth Dependency for UI Routes ---
def verify_ui_auth(request: Request):
    """Protects frontend API routes if a password is set."""
    if DASHBOARD_PASSWORD:
        cookie = request.cookies.get(AUTH_COOKIE_NAME)
        if cookie != EXPECTED_HASH:
            raise HTTPException(status_code=401, detail="Unauthorized")

# --- API Endpoints (Frontend - Protected by Cookie) ---
@app.get("/api/v1/system/state", dependencies=[Depends(verify_ui_auth)])
async def get_system_state():
    conn = sqlite3.connect("/data/nanotrap.db")
    c = conn.cursor()
    c.execute("SELECT value FROM config WHERE key='is_armed'")
    val = c.fetchone()[0]
    conn.close()
    return {"is_armed": val == 'true'}

@app.patch("/api/v1/system/state", dependencies=[Depends(verify_ui_auth)])
async def set_system_state(state: SystemState):
    conn = sqlite3.connect("/data/nanotrap.db")
    c = conn.cursor()
    c.execute("UPDATE config SET value=? WHERE key='is_armed'", ('true' if state.is_armed else 'false',))
    conn.commit()
    conn.close()
    return {"status": "success", "is_armed": state.is_armed}

@app.get("/api/v1/sensors", dependencies=[Depends(verify_ui_auth)])
async def get_sensors():
    conn = sqlite3.connect("/data/nanotrap.db")
    c = conn.cursor()
    c.execute("SELECT sensor_id, sensor_ip, last_seen FROM sensors ORDER BY sensor_id")
    rows = c.fetchall()
    conn.close()
    
    fleet = []
    now = datetime.now()
    for r in rows:
        last_seen_dt = datetime.strptime(r[2], "%Y-%m-%d %H:%M:%S")
        is_online = (now - last_seen_dt) < timedelta(seconds=90)
        fleet.append({
            "sensor_id": r[0], "sensor_ip": r[1], "last_seen": r[2],
            "status": "online" if is_online else "offline"
        })
    return fleet

@app.get("/api/v1/events", dependencies=[Depends(verify_ui_auth)])
async def get_events():
    conn = sqlite3.connect("/data/nanotrap.db")
    c = conn.cursor()
    c.execute("SELECT * FROM events ORDER BY id DESC")
    rows = c.fetchall()
    conn.close()
    
    events = []
    for r in rows:
        events.append({
            "id": r[0], "timestamp": r[1], "sensor_id": r[2], "sensor_ip": r[3], "attacker_ip": r[4],
            "target_port": r[5], "protocol": r[6], "action": r[7], 
            "duration_sec": r[8], "raw_payload": r[9], "is_read": r[10]
        })
    return events

@app.patch("/api/v1/events/read", dependencies=[Depends(verify_ui_auth)])
async def mark_events_read():
    conn = sqlite3.connect("/data/nanotrap.db")
    c = conn.cursor()
    c.execute("UPDATE events SET is_read = 1 WHERE is_read = 0")
    conn.commit()
    conn.close()
    return {"status": "success"}

@app.delete("/api/v1/events", dependencies=[Depends(verify_ui_auth)])
async def clear_events():
    conn = sqlite3.connect("/data/nanotrap.db")
    c = conn.cursor()
    c.execute("DELETE FROM events")
    conn.commit()
    conn.close()
    return {"status": "success"}


# --- API Endpoints (Agents - Protected by x-api-key) ---
@app.post("/api/v1/heartbeat")
async def receive_heartbeat(hb: Heartbeat, x_api_key: str = Header(None)):
    if x_api_key != API_SECRET:
        raise HTTPException(status_code=401, detail="Unauthorized")
        
    now = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    conn = sqlite3.connect("/data/nanotrap.db")
    c = conn.cursor()
    c.execute('''INSERT INTO sensors (sensor_id, sensor_ip, last_seen) 
                 VALUES (?, ?, ?) 
                 ON CONFLICT(sensor_id) DO UPDATE SET last_seen=?, sensor_ip=?''', 
              (hb.sensor_id, hb.sensor_ip, now, now, hb.sensor_ip))
    conn.commit()
    conn.close()
    return {"status": "alive"}

@app.post("/api/v1/event")
async def receive_event(event: Event, x_api_key: str = Header(None)):
    if x_api_key != API_SECRET:
        raise HTTPException(status_code=401, detail="Unauthorized")
    
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    
    if isinstance(event.raw_payload, (list, dict)):
        payload_data = json.dumps(event.raw_payload)
    else:
        payload_data = str(event.raw_payload)
    
    conn = sqlite3.connect("/data/nanotrap.db")
    c = conn.cursor()
    c.execute('''INSERT INTO events 
                 (timestamp, sensor_id, sensor_ip, attacker_ip, target_port, protocol, action, duration_sec, raw_payload, is_read)
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)''', 
              (timestamp, event.sensor_id, event.sensor_ip, event.attacker_ip, event.target_port, 
               event.protocol, event.action, event.duration_sec, payload_data))
    c.execute("SELECT value FROM config WHERE key='is_armed'")
    is_armed = c.fetchone()[0] == 'true'
    conn.commit()
    conn.close()

    if is_armed:
        msg = f"🚨 HoneyWire [{event.sensor_id}]: IP {event.attacker_ip} hit Port {event.target_port}"
        
        if NTFY_URL:
            try: requests.post(NTFY_URL, data=msg.encode('utf-8'))
            except: pass 
                
        if GOTIFY_URL and GOTIFY_TOKEN:
            try:
                requests.post(
                    GOTIFY_URL, headers={"X-Gotify-Key": GOTIFY_TOKEN},
                    json={"title": "HoneyWire Alert", "message": msg, "priority": 5}
                )
            except: pass

    return {"status": "success"}

# --- Web UI & Vault Door ---
LOGIN_HTML = """
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>HoneyWire Vault</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-900 h-screen flex items-center justify-center">
    <div class="bg-gray-800 p-8 rounded-lg shadow-lg border border-gray-700 w-96 text-center">
        <h1 class="text-3xl font-bold text-green-500 mb-6">🕸️ HoneyWire</h1>
        <form onsubmit="doLogin(event)" class="flex flex-col gap-4">
            <input type="password" id="pwd" placeholder="Dashboard Password" class="p-3 rounded bg-gray-900 border border-gray-700 text-white focus:outline-none focus:border-green-500" required>
            <button type="submit" class="bg-green-600 hover:bg-green-500 text-white font-bold py-3 rounded transition-colors">Unlock Vault</button>
        </form>
        <p id="err" class="text-red-500 mt-4 hidden">Access Denied</p>
    </div>
    <script>
        async function doLogin(e) {
            e.preventDefault();
            const res = await fetch('/login', {
                method: 'POST', 
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({password: document.getElementById('pwd').value})
            });
            if(res.ok) window.location.reload();
            else document.getElementById('err').classList.remove('hidden');
        }
    </script>
</body>
</html>
"""

@app.post("/login")
async def login(req: LoginRequest):
    if DASHBOARD_PASSWORD and req.password == DASHBOARD_PASSWORD:
        response = JSONResponse(content={"status": "ok"})
        # Set a 30-day secure cookie
        response.set_cookie(key=AUTH_COOKIE_NAME, value=EXPECTED_HASH, max_age=2592000, httponly=True)
        return response
    raise HTTPException(status_code=401, detail="Invalid Password")

@app.get("/logout")
async def logout():
    response = HTMLResponse(content="<script>window.location.href='/';</script>")
    response.delete_cookie(AUTH_COOKIE_NAME)
    return response

@app.get("/", response_class=HTMLResponse)
async def serve_dashboard(request: Request):
    # Check the Vault Door
    if DASHBOARD_PASSWORD:
        cookie = request.cookies.get(AUTH_COOKIE_NAME)
        if cookie != EXPECTED_HASH:
            return HTMLResponse(content=LOGIN_HTML)
            
    return templates.TemplateResponse(request=request, name="index.html")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)