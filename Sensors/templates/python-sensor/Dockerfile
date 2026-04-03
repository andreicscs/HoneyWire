# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

# Install git (Required to fetch the SDK from GitHub)
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

# Create virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Install requirements (This pulls the SDK + your custom libraries)
COPY requirements.txt .
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# --- Stage 2: Distroless (Production) ---
FROM gcr.io/distroless/python3-debian12
WORKDIR /app

# Copy the compiled virtual environment from the builder
COPY --from=builder /opt/venv /opt/venv

ENV PYTHONPATH="/opt/venv/lib/python3.11/site-packages"
ENV PATH="/opt/venv/bin:$PATH"
ENV PYTHONUNBUFFERED=1

# Copy the custom sensor script
COPY sensor.py .

# Run the sensor
CMD ["sensor.py"]