# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

COPY SDKs/python-honeywire /tmp/sdk_src
RUN pip install --no-cache-dir wheel && \
    pip wheel /tmp/sdk_src --wheel-dir /tmp/wheels

COPY Sensors/official/IcmpCanary/requirements.txt .
RUN pip install --no-cache-dir --target /app/lib /tmp/wheels/honeywire-*.whl -r requirements.txt


# --- Stage 2: Runtime (Debian Slim) ---
FROM python:3.11-slim
WORKDIR /app

# Upgrade to patch known vulnerabilities, then install native libpcap
RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y libpcap0.8 && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/lib /app/lib
COPY Sensors/official/IcmpCanary/IcmpCanary.py .

ENV PYTHONPATH="/app/lib"
ENV PYTHONUNBUFFERED=1

CMD ["python3", "IcmpCanary.py"]