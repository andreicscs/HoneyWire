# --- Stage 1: Builder ---
FROM python:3.11-alpine AS builder
WORKDIR /app

# Install Alpine build tools
RUN apk add --no-cache git gcc musl-dev linux-headers libpcap-dev

# 1. Copy the SDK source
COPY SDKs/python-honeywire /tmp/sdk_src

# 2. Build the SDK wheel
RUN pip install --no-cache-dir wheel && \
    pip wheel /tmp/sdk_src --wheel-dir /tmp/wheels

# 3. Install dependencies
COPY Sensors/official/IcmpCanary/requirements.txt .
RUN pip install --no-cache-dir --target /app/lib /tmp/wheels/honeywire-*.whl -r requirements.txt


# --- Stage 2: Runtime (Zero Vulnerability Alpine) ---
FROM python:3.11-alpine
WORKDIR /app

# Install libpcap natively so it has all its required dependencies!
RUN apk add --no-cache libpcap

COPY --from=builder /app/lib /app/lib
COPY Sensors/official/IcmpCanary/PingCanary.py .

ENV PYTHONPATH="/app/lib"
ENV PYTHONUNBUFFERED=1

# Explicitly call python3 to avoid Error 127
CMD ["python3", "PingCanary.py"]