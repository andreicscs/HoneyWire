import sqlite3
import os
import requests
import json
from fastapi import FastAPI, Header, HTTPException, Request
from fastapi.responses import HTMLResponse
from fastapi.templating import Jinja2Templates
from pydantic import BaseModel
from typing import Any
from datetime import datetime, timedelta

# --- Configuration ---
API_SECRET = os.getenv("API_SECRET", "super_secret_key_123")
NTFY_URL = os.getenv("NTFY_URL", "")
# Add the Gotify Environment Variables
GOTIFY_URL = os.getenv("GOTIFY_URL", "")
GOTIFY_TOKEN = os.getenv("GOTIFY_TOKEN", "")

app = FastAPI(title="HoneyWire Hub")
templates = Jinja2Templates(directory="templates")

# --- Database Setup ---
def init_db():
    conn = sqlite3.connect("nanotrap.db")
    c = conn.cursor()
    c.execute('''CREATE TABLE IF NOT EXISTS events
                 (id INTEGER PRIMARY KEY AUTOINCREMENT,
                  timestamp TEXT, sensor_id TEXT, sensor_ip TEXT, attacker_ip TEXT, 
                  target_port INTEGER, protocol TEXT, action TEXT, 
                  duration_sec REAL, raw_payload TEXT, is_read INTEGER DEFAULT 0)''')
    c.execute('''CREATE TABLE IF NOT EXISTS sensors
                 (sensor_id TEXT PRIMARY KEY, sensor_ip TEXT, last_seen TEXT)''')
    c.execute('''CREATE TABLE IF NOT EXISTS config (key TEXT PRIMARY KEY, value TEXT)''')
    # Default to Armed
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
    # Use 'Any' so Pydantic stops blocking JSON arrays
    raw_payload: Any = ""

class Heartbeat(BaseModel):
    sensor_id: str
    sensor_ip: str

class SystemState(BaseModel):
    is_armed: bool

# --- API Endpoints ---
@app.get("/api/v1/system/state")
async def get_system_state():
    conn = sqlite3.connect("nanotrap.db")
    c = conn.cursor()
    c.execute("SELECT value FROM config WHERE key='is_armed'")
    val = c.fetchone()[0]
    conn.close()
    return {"is_armed": val == 'true'}

@app.patch("/api/v1/system/state")
async def set_system_state(state: SystemState):
    conn = sqlite3.connect("nanotrap.db")
    c = conn.cursor()
    c.execute("UPDATE config SET value=? WHERE key='is_armed'", ('true' if state.is_armed else 'false',))
    conn.commit()
    conn.close()
    return {"status": "success", "is_armed": state.is_armed}

@app.post("/api/v1/heartbeat")
async def receive_heartbeat(hb: Heartbeat, x_api_key: str = Header(None)):
    """The Agent calls this every 30 seconds to say 'I am alive'"""
    if x_api_key != API_SECRET:
        raise HTTPException(status_code=401, detail="Unauthorized")
        
    now = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    conn = sqlite3.connect("nanotrap.db")
    c = conn.cursor()
    # Upsert the sensor's last_seen time
    c.execute('''INSERT INTO sensors (sensor_id, sensor_ip, last_seen) 
                 VALUES (?, ?, ?) 
                 ON CONFLICT(sensor_id) DO UPDATE SET last_seen=?, sensor_ip=?''', 
              (hb.sensor_id, hb.sensor_ip, now, now, hb.sensor_ip))
    conn.commit()
    conn.close()
    return {"status": "alive"}

@app.get("/api/v1/sensors")
async def get_sensors():
    """Returns the fleet and calculates if they are offline"""
    conn = sqlite3.connect("nanotrap.db")
    c = conn.cursor()
    c.execute("SELECT sensor_id, sensor_ip, last_seen FROM sensors ORDER BY sensor_id")
    rows = c.fetchall()
    conn.close()
    
    fleet = []
    now = datetime.now()
    for r in rows:
        last_seen_dt = datetime.strptime(r[2], "%Y-%m-%d %H:%M:%S")
        # If no ping in 90 seconds, mark as offline
        is_online = (now - last_seen_dt) < timedelta(seconds=90)
        
        fleet.append({
            "sensor_id": r[0],
            "sensor_ip": r[1],
            "last_seen": r[2],
            "status": "online" if is_online else "offline"
        })
    return fleet

@app.post("/api/v1/event")
async def receive_event(event: Event, x_api_key: str = Header(None)):
    if x_api_key != API_SECRET:
        raise HTTPException(status_code=401, detail="Unauthorized")
    
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    
    # Bulletproof payload conversion
    if isinstance(event.raw_payload, (list, dict)):
        payload_data = json.dumps(event.raw_payload)
    else:
        payload_data = str(event.raw_payload)
    
    conn = sqlite3.connect("nanotrap.db")
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
        
        # --- NTFY Notification ---
        if NTFY_URL:
            try:
                requests.post(NTFY_URL, data=msg.encode('utf-8'))
            except:
                pass 
                
        # --- GOTIFY Notification ---
        if GOTIFY_URL and GOTIFY_TOKEN:
            try:
                requests.post(
                    GOTIFY_URL,
                    headers={"X-Gotify-Key": GOTIFY_TOKEN},
                    json={
                        "title": "HoneyWire Alert",
                        "message": msg,
                        "priority": 5 # Priority 5 triggers a push notification with sound in the app
                    }
                )
            except:
                pass

    return {"status": "success"}

@app.get("/api/v1/events")
async def get_events():
    conn = sqlite3.connect("nanotrap.db")
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

@app.patch("/api/v1/events/read")
async def mark_events_read():
    conn = sqlite3.connect("nanotrap.db")
    c = conn.cursor()
    c.execute("UPDATE events SET is_read = 1 WHERE is_read = 0")
    conn.commit()
    conn.close()
    return {"status": "success"}

@app.delete("/api/v1/events")
async def clear_events():
    conn = sqlite3.connect("nanotrap.db")
    c = conn.cursor()
    c.execute("DELETE FROM events")
    conn.commit()
    conn.close()
    return {"status": "success"}

# --- Web UI ---
@app.get("/", response_class=HTMLResponse)
async def serve_dashboard(request: Request):
    return templates.TemplateResponse(request=request, name="index.html")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)