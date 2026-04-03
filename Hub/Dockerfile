# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

# 1. Patch the OS (This is why your Debian vulnerabilities are now 0!)
RUN apt-get update && apt-get upgrade -y && \
    rm -rf /var/lib/apt/lists/*

# 2. Create the virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# 3. CRITICAL: Upgrade core tools to fix the 'jaraco' and 'wheel' vulnerabilities
# This pulls the fixed versions (6.1.0+ and 0.46.2+) identified in your scan
RUN pip install --no-cache-dir --upgrade pip setuptools wheel

# 4. Install your pinned requirements
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 5. Prepare data directory for Distroless
RUN mkdir /data && chown 65532:65532 /data

# --- Stage 2: Distroless & Rootless ---
FROM gcr.io/distroless/python3-debian12
WORKDIR /app

COPY --from=builder /opt/venv /opt/venv
ENV PYTHONPATH="/opt/venv/lib/python3.11/site-packages"
ENV PATH="/opt/venv/bin:$PATH"

COPY --from=builder --chown=65532:65532 /data /data
COPY --chown=65532:65532 main.py /app/main.py
COPY --chown=65532:65532 templates/ /app/templates/

USER nonroot:nonroot
EXPOSE 8080
CMD ["/app/main.py"]