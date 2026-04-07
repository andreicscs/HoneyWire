# STAGE 1: Build
FROM golang:1.25 AS builder

WORKDIR /src
COPY SDKs/go-honeywire ./SDKs/go-honeywire
COPY Sensors/official/WebRouterDecoy ./Sensors/official/WebRouterDecoy

WORKDIR /src/Sensors/official/WebRouterDecoy
RUN go mod download

# Pure Go, fully static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o web-decoy ./main.go

# STAGE 2: Distroless
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /src/Sensors/official/WebRouterDecoy/web-decoy /app/web-decoy

USER 65532
CMD ["/app/web-decoy"]