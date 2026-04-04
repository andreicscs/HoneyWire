# Sensors/templates/python-sensor/Dockerfile

# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

# Install git (Required to fetch the SDK from GitHub)
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

# Copy requirements (This pulls the SDK + your custom libraries)
COPY requirements.txt .

# Install dependencies into a flat directory (--target /app/lib)
# This avoids virtualenv symlinks which can break inside Distroless
RUN pip install --no-cache-dir --target /app/lib -r requirements.txt

# --- Stage 2: Distroless (Production) ---
FROM gcr.io/distroless/python3-debian12
WORKDIR /app

# Copy the FLAT library folder. No symlinks, just code.
COPY --from=builder /app/lib /app/lib

# Copy the custom sensor script
COPY sensor.py .

# Tell Python to look in our lib folder first
ENV PYTHONPATH="/app/lib"
ENV PYTHONUNBUFFERED=1

# Distroless runs /usr/bin/python3 by default
CMD ["sensor.py"]