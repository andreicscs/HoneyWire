# --- Stage 1: Builder ---
FROM python:3.11-slim AS builder
WORKDIR /app

# Create a virtual environment and install dependencies
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Trick: Create the data directory here and give it to the Distroless nonroot user (UID 65532)
RUN mkdir /data && chown 65532:65532 /data

# --- Stage 2: Distroless & Rootless (Production) ---
FROM gcr.io/distroless/python3-debian12

WORKDIR /app

# Copy the virtual environment
COPY --from=builder /opt/venv /opt/venv
ENV PYTHONPATH="/opt/venv/lib/python3.11/site-packages"
ENV PATH="/opt/venv/bin:$PATH"

# Copy the pre-chowned data directory from the builder
COPY --from=builder --chown=65532:65532 /data /data

# Copy the app files
COPY --chown=65532:65532 main.py /app/main.py
COPY --chown=65532:65532 templates/ /app/templates/

# Drop privileges to the non-root user
USER nonroot:nonroot

EXPOSE 8080
CMD ["main.py"]