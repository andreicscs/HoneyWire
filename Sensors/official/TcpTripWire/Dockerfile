# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

# 1. Copy the SDK source
COPY SDKs/python-honeywire /tmp/sdk_src

# 2. Build a WHEEL (a static redistribution file) instead of installing directly.
# This ensures all code is physically packaged and not symlinked.
RUN pip install --no-cache-dir wheel && \
    pip wheel /tmp/sdk_src --wheel-dir /tmp/wheels

# 3. Install the wheel and the requirements into a specific folder
# We use --target to ensure everything is in one flat, movable directory.
COPY Sensors/official/TcpTripWire/requirements.txt .
RUN pip install --no-cache-dir --target /app/lib /tmp/wheels/honeywire-*.whl -r requirements.txt

# --- Stage 2: Distroless (Production) ---
FROM gcr.io/distroless/python3-debian12
WORKDIR /app

# 4. Copy the FLAT library folder. No symlinks, just code.
COPY --from=builder /app/lib /app/lib

# 5. Copy the sensor script
COPY Sensors/official/TcpTripWire/TcpTripWire.py .

# 6. Tell Python to look in our lib folder first.
ENV PYTHONPATH="/app/lib"
ENV PYTHONUNBUFFERED=1 

# Distroless runs /usr/bin/python3 by default
CMD ["TcpTripWire.py"]