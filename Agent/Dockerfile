# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

# Create a virtual environment and install dependencies
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
RUN pip install --no-cache-dir requests

# --- Stage 2: Distroless (Production) ---
FROM gcr.io/distroless/python3-debian12
WORKDIR /app

# Copy the virtual environment from the builder
COPY --from=builder /opt/venv /opt/venv

ENV PYTHONPATH="/opt/venv/lib/python3.11/site-packages"
ENV PATH="/opt/venv/bin:$PATH"

# Copy your agent script
COPY agent.py .

# Notice: We intentionally do NOT use a nonroot user here.
# We remain 'root' to allow binding to privileged decoy ports (1-1024).

CMD ["agent.py"]