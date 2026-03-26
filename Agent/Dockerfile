FROM python:3.11-alpine
WORKDIR /app
RUN pip install --no-cache-dir requests
COPY agent.py .
CMD ["python", "agent.py"]