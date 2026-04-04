# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

# 1. Copy the SDK source from the repository root.
COPY SDKs/python-honeywire /tmp/sdk_src

# 2. Build a wheel for the SDK.
RUN pip install --no-cache-dir wheel && \
    pip wheel /tmp/sdk_src --wheel-dir /tmp/wheels

# 3. Install the wheel and sensor-specific dependencies.
COPY Sensors/official/NetworkScanDetector/requirements.txt .
RUN pip install --no-cache-dir --target /app/lib /tmp/wheels/honeywire-*.whl -r requirements.txt

# --- Stage 2: Runtime ---
FROM python:3.11-slim
WORKDIR /app

RUN apt-get update && apt-get install -y libpcap0.8 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/lib /app/lib
COPY Sensors/official/NetworkScanDetector/NetworkScanDetector.py .

ENV PYTHONPATH="/app/lib"
ENV PYTHONUNBUFFERED=1

CMD ["python3", "NetworkScanDetector.py"]
