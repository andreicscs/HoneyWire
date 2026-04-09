# STAGE 1: Frontend Builder
FROM node:20-alpine AS frontend-builder
WORKDIR /app/ui

# Install dependencies
COPY ui/package*.json ./
RUN npm ci

# Copy source and build the Vue SPA
COPY ui/ ./
RUN npm run build

# STAGE 2: Go Builder
FROM golang:1.25 AS go-builder
WORKDIR /build

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the Go source code
COPY . .

# CRITICAL: Copy the compiled Vue app from Stage 1 into the Go build context
# so the `//go:embed all:dist` directive can find it.
COPY --from=frontend-builder /app/ui/dist ./ui/dist

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o honeywire-hub ./cmd/hub/main.go

# STAGE 3: Distroless Production Image
# The "static" image contains literally nothing except SSL certificates and a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=go-builder /build/honeywire-hub /app/

EXPOSE 8080
CMD ["/app/honeywire-hub"]