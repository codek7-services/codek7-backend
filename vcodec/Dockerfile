# Multi-stage build for Rust video processor with FFmpeg
FROM rust:1.82-slim AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy manifests
COPY vcodec/Cargo.toml vcodec/Cargo.lock ./
COPY vcodec/build.rs ./

# Copy source code
COPY vcodec/src/ ./src/
COPY ../common/pb/ ./common/pb/ 

# Build the application in release mode
RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies including FFmpeg
RUN apt-get update && apt-get install -y \
    ffmpeg \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create app user for security
RUN groupadd -r appuser && useradd -r -g appuser appuser

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/target/release/vcodec /usr/local/bin/vcodec

# Create directories for processing
RUN mkdir -p /app/input /app/output /app/temp && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port (adjust as needed)
EXPOSE 9600 

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ffmpeg -version > /dev/null || exit 1

# Set environment variables
ENV RUST_LOG=info
ENV RUST_BACKTRACE=1

# Default command
CMD ["vcodec"]
