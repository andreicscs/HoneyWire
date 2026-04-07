# STAGE 1: Build
FROM golang:1.25 AS builder

WORKDIR /src
COPY SDKs/go-honeywire ./SDKs/go-honeywire
COPY Sensors/official/IcmpCanary-GO ./Sensors/official/IcmpCanary-GO

WORKDIR /src/Sensors/official/IcmpCanary-GO
RUN go mod download

# Pure Go, fully static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o icmp-canary ./main.go

# STAGE 2: Distroless
# We use the standard image instead of :nonroot because raw sockets require UID 0
FROM gcr.io/distroless/static-debian12:latest

COPY --from=builder /src/Sensors/official/IcmpCanary-GO/icmp-canary /app/icmp-canary

CMD ["/app/icmp-canary"]