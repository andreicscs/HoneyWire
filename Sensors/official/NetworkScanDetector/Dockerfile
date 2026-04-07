# STAGE 1: Build
FROM golang:1.25 AS builder

WORKDIR /src
COPY SDKs/go-honeywire ./SDKs/go-honeywire
COPY Sensors/official/NetworkScanDetector ./Sensors/official/NetworkScanDetector

WORKDIR /src/Sensors/official/NetworkScanDetector
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o scan-detector ./main.go

# STAGE 2: Distroless
FROM gcr.io/distroless/static-debian12:latest

COPY --from=builder /src/Sensors/official/NetworkScanDetector/scan-detector /app/scan-detector

CMD ["/app/scan-detector"]