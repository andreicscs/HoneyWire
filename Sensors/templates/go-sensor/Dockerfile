# STAGE 1: Build
FROM golang:1.25 AS builder

WORKDIR /src

# 1. Copy the shared SDK into the builder
COPY SDKs/go-honeywire ./SDKs/go-honeywire

# 2. Copy the specific sensor code (Change 'custom/MySensor' to your folder name)
COPY Sensors/custom/MySensor ./Sensors/custom/MySensor

# 3. Move into the sensor folder to build
WORKDIR /src/Sensors/custom/MySensor

RUN go mod download

# Compile the pure Go, fully static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o custom-sensor ./main.go

# STAGE 2: Distroless Sandbox
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /src/Sensors/custom/MySensor/custom-sensor /app/custom-sensor

# Run as an unprivileged user
USER 65532
CMD ["/app/custom-sensor"]