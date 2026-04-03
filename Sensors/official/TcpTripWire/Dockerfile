# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

# 1. Install git (Required for pip to pull the HoneyWire SDK from GitHub)
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

# 2. Create a virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# 3. Copy requirements and install
# (This fetches requests, aiohttp, and the SDK directly into the venv)
COPY requirements.txt .
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# --- Stage 2: Distroless (Production) ---
FROM gcr.io/distroless/python3-debian12
WORKDIR /app

# 1. Copy the compiled virtual environment from the builder
COPY --from=builder /opt/venv /opt/venv

# 2. Set environment variables so Python knows where the packages are
ENV PYTHONPATH="/opt/venv/lib/python3.11/site-packages"
ENV PATH="/opt/venv/bin:$PATH"
# Ensures Python print() statements go straight to Docker logs without delay
ENV PYTHONUNBUFFERED=1 

# 3. Copy your specific sensor script
COPY TcpTripWire.py .

# Notice: We intentionally do NOT use a nonroot user here.
# We remain 'root' to allow binding to privileged decoy ports (e.g., Port 22, 80).
CMD ["TcpTripWire.py"]