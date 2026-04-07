# STAGE 1: Build
FROM golang:1.25 AS builder

WORKDIR /src
COPY SDKs/go-honeywire ./SDKs/go-honeywire
COPY Sensors/official/NetworkScanDetector-GO ./Sensors/official/NetworkScanDetector-GO

WORKDIR /src/Sensors/official/NetworkScanDetector-GO
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o scan-detector ./main.go

# STAGE 2: Distroless
FROM gcr.io/distroless/static-debian12:latest

COPY --from=builder /src/Sensors/official/NetworkScanDetector-GO/scan-detector /app/scan-detector

CMD ["/app/scan-detector"]