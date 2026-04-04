# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

RUN apt-get update && apt-get install -y git libpcap0.8 libpcap-dev && rm -rf /var/lib/apt/lists/*

# 1. Create the clibs folder and copy the versioned library
RUN mkdir -p /app/clibs && \
    cp /usr/lib/*-linux-gnu/libpcap.so.0.8 /app/clibs/ && \
    # 2. MANDATORY: Create the generic symlink Scapy expects
    ln -s /app/clibs/libpcap.so.0.8 /app/clibs/libpcap.so

# 3. Copy SDK and build
COPY SDKs/python-honeywire /tmp/sdk_src
RUN pip install --no-cache-dir wheel && \
    pip wheel /tmp/sdk_src --wheel-dir /tmp/wheels

# 4. Install dependencies to the flat /app/lib folder
COPY Sensors/official/NetworkScanDetector/requirements.txt .
RUN pip install --no-cache-dir --target /app/lib /tmp/wheels/honeywire-*.whl -r requirements.txt

# --- Stage 2: Distroless ---
FROM gcr.io/distroless/python3-debian12
WORKDIR /app

# 3. Copy the clibs (including the symlink)
COPY --from=builder /app/clibs /app/clibs

# 4. Same as before
COPY --from=builder /app/lib /app/lib
COPY Sensors/official/NetworkScanDetector/NetworkScanDetector.py .

ENV PYTHONPATH="/app/lib"
# 5. Tell the system to look in /app/clibs for our stolen files
ENV LD_LIBRARY_PATH="/app/clibs"
ENV PYTHONUNBUFFERED=1

CMD ["NetworkScanDetector.py"]