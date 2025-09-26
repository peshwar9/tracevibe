# Multi-stage build for TraceVibe
# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install git, gcc, and other dependencies needed for CGO
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application for multiple architectures
# The TARGETPLATFORM arg is automatically set by Docker buildx
ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN echo "Building for $TARGETPLATFORM on $BUILDPLATFORM"

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -o tracevibe .

# Stage 2: Create the final lightweight image
FROM alpine:latest

# Install ca-certificates, glibc compatibility and SQLite runtime
RUN apk --no-cache add ca-certificates tzdata libc6-compat

# Create app directory
WORKDIR /app

# Create a non-root user
RUN addgroup -g 1000 -S tracevibe && \
    adduser -u 1000 -S tracevibe -G tracevibe

# Copy the binary from builder stage
COPY --from=builder /app/tracevibe .

# Copy any static assets if needed
COPY --from=builder /app/cmd/web/templates ./cmd/web/templates

# Create data directory for SQLite database
RUN mkdir -p /app/data && chown -R tracevibe:tracevibe /app

# Switch to non-root user
USER tracevibe

# Expose port 8080 (default port)
EXPOSE 8080

# Set environment variables
ENV GIN_MODE=release
ENV DB_PATH=/app/data/tracevibe.db

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Start the application
CMD ["./tracevibe", "serve", "--port", "8080"]