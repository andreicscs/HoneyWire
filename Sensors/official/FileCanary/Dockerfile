# STAGE 1: Build
FROM golang:1.25 AS builder

WORKDIR /src

# 1. Copy the shared SDK into the builder
COPY SDKs/go-honeywire ./SDKs/go-honeywire

# 2. Copy the specific sensor code
COPY Sensors/official/FileCanary-GO ./Sensors/official/FileCanary-GO

# 3. Move into the sensor folder to build
WORKDIR /src/Sensors/official/FileCanary-GO

# Download dependencies (this will now successfully find the local SDK)
RUN go mod download

# Compile the pure Go, fully static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o file-canary ./main.go

# STAGE 2: Distroless
FROM gcr.io/distroless/static-debian12:nonroot

# Create a default mount point with correct nonroot ownership
COPY --chown=65532:65532 --from=builder /src/Sensors/official/FileCanary-GO/file-canary /app/file-canary

USER 65532
CMD ["/app/file-canary"]