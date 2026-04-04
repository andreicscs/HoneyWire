"""
HoneyWire Official Sensor: Web Router Decoy
Sensor type: web_honeypot

This sensor serves a deceptive router administration login page and captures credentials when an attacker submits the form.
"""

import asyncio
import os
from typing import Dict

from fastapi import FastAPI, Form, Request, status
from fastapi.responses import HTMLResponse, PlainTextResponse
from fastapi.middleware.cors import CORSMiddleware
from honeywire import HoneyWireSensor
import uvicorn


LOGIN_PAGE = """
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Router Login</title>
  <style>
    body { background: #f1f5f9; color: #111827; font-family: Inter, system-ui, sans-serif; }
    .card { max-width: 420px; margin: 6rem auto; padding: 2rem; background: white; box-shadow: 0 24px 80px rgba(15,23,42,0.08); border-radius: 1rem; }
    h1 { margin-bottom: 1.25rem; font-size: 1.75rem; letter-spacing: -.03em; }
    label { display: block; margin-top: 1rem; font-size: 0.9rem; color: #374151; }
    input { width: 100%; margin-top: 0.5rem; padding: 0.9rem 1rem; border: 1px solid #d1d5db; border-radius: 0.75rem; }
    button { width: 100%; margin-top: 1.75rem; padding: 0.95rem 1rem; background: #0f766e; color: white; border: none; border-radius: 0.75rem; font-weight: 700; cursor: pointer; }
    .footer { margin-top: 1.5rem; font-size: 0.85rem; color: #6b7280; }
  </style>
</head>
<body>
  <div class="card">
    <h1>Router Login</h1>
    <p>Sign in to the router administration panel.</p>
    <form method="post" action="/login">
      <label>Username</label>
      <input name="username" type="text" autocomplete="username" value="admin" />
      <label>Password</label>
      <input name="password" type="password" autocomplete="current-password" />
      <button type="submit">Login</button>
    </form>
    <div class="footer">If credentials are invalid, please try again.</div>
  </div>
</body>
</html>
"""


class WebRouterDecoy(HoneyWireSensor):
    def __init__(self):
        super().__init__(sensor_type="web_honeypot")
        self.port = int(os.getenv("HW_BIND_PORT", "8080"))
        self.router_brand = os.getenv("HW_ROUTER_BRAND", "Netgear")
        self.app = FastAPI()

        self.app.add_middleware(
            CORSMiddleware,
            allow_origins=["*"],
            allow_methods=["GET", "POST"],
            allow_headers=["*"]
        )

        self._register_routes()

    def _register_routes(self):
        @self.app.get("/", response_class=HTMLResponse)
        async def root(request: Request):
            return HTMLResponse(LOGIN_PAGE)

        @self.app.post("/login", response_class=HTMLResponse)
        async def login(request: Request, username: str = Form(""), password: str = Form("")):
            user_agent = request.headers.get("user-agent", "Unknown")
            source_ip = request.client.host if request.client else "Unknown"

            metadata = {
                "user_agent": user_agent,
                "attempted_username": username,
                "attempted_password": password,
                "remote_ip": source_ip,
            }

            await asyncio.to_thread(
                self.report_event,
                event_type="web_login_attempt",
                severity="critical",
                metadata=metadata,
                action_taken="logged",
                source=source_ip,
                target="Web Interface",
            )

            return HTMLResponse(
                content="<h1>401 Unauthorized</h1><p>Invalid Username or Password.</p>",
                status_code=status.HTTP_401_UNAUTHORIZED,
            )

    async def monitor(self):
        print(f"[*] HoneyWire Web Router Decoy Sensor | Port {self.port} | Severity: {self.severity}")
        config = uvicorn.Config(self.app, host="0.0.0.0", port=self.port, log_level="warning")
        server = uvicorn.Server(config)
        await server.serve()


if __name__ == "__main__":
    sensor = WebRouterDecoy()
    try:
        asyncio.run(sensor.start())
    except KeyboardInterrupt:
        print("\n[*] Shutting down Web Router Decoy sensor...")
