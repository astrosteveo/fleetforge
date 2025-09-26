# FleetForge Multi-Stage Dockerfile

# Stage 1: Build stage
FROM golang:1.25-alpine AS build

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fleetforge ./cmd/cell

# Stage 2: Production stage
FROM alpine:3.18 AS production

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -S fleetforge && adduser -S fleetforge -G fleetforge

# Set working directory
WORKDIR /app

# Copy binary from build stage
COPY --from=build /build/fleetforge .

# Change ownership to non-root user
RUN chown -R fleetforge:fleetforge /app

# Switch to non-root user
USER fleetforge

# Expose default port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Start the application
CMD ["./fleetforge"]