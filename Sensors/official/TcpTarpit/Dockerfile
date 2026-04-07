# STAGE 1: Build
FROM golang:1.25 AS builder

WORKDIR /src
COPY SDKs/go-honeywire ./SDKs/go-honeywire
COPY Sensors/official/TcpTarpit-GO ./Sensors/official/TcpTarpit-GO

WORKDIR /src/Sensors/official/TcpTarpit-GO
RUN go mod download

# Pure Go, fully static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o tcp-tarpit ./main.go

# STAGE 2: Distroless
# Back to the fully unprivileged nonroot container!
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /src/Sensors/official/TcpTarpit-GO/tcp-tarpit /app/tcp-tarpit

USER 65532
CMD ["/app/tcp-tarpit"]