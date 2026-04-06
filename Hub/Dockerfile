# STAGE 1: Builder
FROM golang:1.25 AS builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o honeywire-hub ./cmd/hub/main.go

# STAGE 2: Distroless
# The "static" image contains literally nothing except SSL certificates and a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /build/honeywire-hub /app/

# Expose port and run
EXPOSE 8080
CMD ["/app/honeywire-hub"]