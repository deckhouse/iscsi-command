# ==========================
# Stage 1: Multi-Architecture Build
# ==========================
FROM --platform=$BUILDPLATFORM golang:1.23.2 AS builder

# Set up working directory
WORKDIR /app

# Copy source code
COPY . .

# Install dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    autoconf automake libtool pkg-config musl-dev gcc make && \
    rm -rf /var/lib/apt/lists/*

# Set up architecture variables
ARG TARGETPLATFORM
RUN echo "Building for $TARGETPLATFORM"

# Установить правильную архитектуру в зависимости от платформы
RUN case "$TARGETPLATFORM" in \
        "linux/amd64")  export GOARCH=amd64 ;; \
        "linux/arm64")  export GOARCH=arm64 ;; \
        *) echo "Unsupported architecture: $TARGETPLATFORM" && exit 1 ;; \
    esac

# Пересобрать libiscsi под ARM64/AMD64
RUN rm -rf libiscsi && git clone --depth 1 https://github.com/deckhouse/libiscsi.git && \
    cd libiscsi && ./autogen.sh && ./configure && make -j$(nproc) && \
    cp utils/iscsi-ls /usr/local/bin/ && chmod +x /usr/local/bin/iscsi-ls

# Build the Go application with correct architecture
RUN mkdir -p /app/build/bin && \
    CGO_ENABLED=1 GOOS=linux GOARCH=$(go env GOARCH) go build -o /app/build/bin/iscsi-server ./cmd/server/main.go

# ==========================
# Stage 2: Minimal Runtime Image
# ==========================
FROM alpine:latest

# Install necessary runtime dependencies
RUN apk add --no-cache ca-certificates musl libc6-compat

# Set up working directory
WORKDIR /root/

# Copy the built binaries
COPY --from=builder /app/build/bin/iscsi-server /usr/local/bin/iscsi-server
COPY --from=builder /usr/local/bin/iscsi-ls /usr/local/bin/iscsi-ls

# Ensure binaries are executable
RUN chmod +x /usr/local/bin/iscsi-server /usr/local/bin/iscsi-ls

# Expose the gRPC port
EXPOSE 50051

# Start the application
ENTRYPOINT ["/usr/local/bin/iscsi-server"]
