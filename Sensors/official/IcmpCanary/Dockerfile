# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

# 1. Install git and the libpcap library
RUN apt-get update && apt-get install -y git libpcap0.8 && rm -rf /var/lib/apt/lists/*

# 2. Extract the compiled libpcap C-libraries into a staging folder
# We use wildcards so this works on both Intel/AMD (x86) and Apple Silicon (ARM)
RUN mkdir -p /app/clibs && cp /usr/lib/*-linux-gnu/libpcap.so.* /app/clibs/

# 3. Copy SDK and build
COPY SDKs/python-honeywire /tmp/sdk_src
RUN pip install --no-cache-dir wheel && \
    pip wheel /tmp/sdk_src --wheel-dir /tmp/wheels

# 4. Install dependencies to the flat /app/lib folder
COPY Sensors/official/IcmpCanary/requirements.txt .
RUN pip install --no-cache-dir --target /app/lib /tmp/wheels/honeywire-*.whl -r requirements.txt


# --- Stage 2: Production (Distroless) ---
FROM gcr.io/distroless/python3-debian12
WORKDIR /app

# 5. Copy the smuggled C-libraries from the builder
COPY --from=builder /app/clibs /app/clibs

# 6. Copy the Python dependencies
COPY --from=builder /app/lib /app/lib

# 7. Copy your sensor logic
COPY Sensors/official/IcmpCanary/PingCanary.py .

# 8. Tell Python where your packages are, AND tell Linux where your smuggled C-libraries are
ENV PYTHONPATH="/app/lib"
ENV LD_LIBRARY_PATH="/app/clibs"
ENV PYTHONUNBUFFERED=1

# Distroless has Python as the hardcoded ENTRYPOINT, so this works natively
CMD ["PingCanary.py"]