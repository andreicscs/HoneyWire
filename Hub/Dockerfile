# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

# 1. Patch the Builder OS to ensure the build environment is secure
RUN apt-get update && apt-get upgrade -y && \
    rm -rf /var/lib/apt/lists/*

# 2. Update core pip tools
RUN pip install --no-cache-dir --upgrade pip setuptools wheel

# 3. Create a virtual environment and install pinned dependencies
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 4. Prepare data directory for the Distroless nonroot user (UID 65532)
RUN mkdir /data && chown 65532:65532 /data

# --- Stage 2: Distroless & Rootless (Production) ---
# We use the official Google Distroless image for the smallest attack surface
FROM gcr.io/distroless/python3-debian12

WORKDIR /app

# Copy the patched virtual environment
COPY --from=builder /opt/venv /opt/venv
ENV PYTHONPATH="/opt/venv/lib/python3.11/site-packages"
ENV PATH="/opt/venv/bin:$PATH"

# Copy the pre-chowned data directory
COPY --from=builder --chown=65532:65532 /data /data

# Copy only necessary app files
COPY --chown=65532:65532 main.py /app/main.py
COPY --chown=65532:65532 templates/ /app/templates/

# Run as non-privileged user
USER nonroot:nonroot

EXPOSE 8080

# Distroless uses the python binary directly
CMD ["/app/main.py"]