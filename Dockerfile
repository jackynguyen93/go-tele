# Multi-stage Dockerfile for Telegram Channel Monitor
# Stage 1: Build TDLib
FROM alpine:3.18 AS tdlib-builder

# Install build dependencies
RUN apk add --no-cache \
    alpine-sdk \
    linux-headers \
    git \
    make \
    cmake \
    gperf \
    openssl-dev \
    zlib-dev \
    readline-dev

# Build TDLib
WORKDIR /tmp
RUN git clone https://github.com/tdlib/td.git && \
    cd td && \
    git checkout master && \
    mkdir build && \
    cd build && \
    cmake -DCMAKE_BUILD_TYPE=Release \
          -DCMAKE_INSTALL_PREFIX=/usr/local \
          .. && \
    cmake --build . -j$(nproc) && \
    cmake --build . --target install

# Stage 2: Build Go application
FROM golang:1.24-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    g++ \
    musl-dev \
    openssl-dev \
    zlib-dev \
    linux-headers

# Copy TDLib from previous stage
COPY --from=tdlib-builder /usr/local/include /usr/local/include
COPY --from=tdlib-builder /usr/local/lib /usr/local/lib

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 \
    CGO_CFLAGS="-I/usr/local/include" \
    CGO_LDFLAGS="-L/usr/local/lib" \
    go build -tags libtdjson -v -o tdclient ./cmd/tdclient

# Stage 3: Runtime
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    openssl \
    zlib \
    libstdc++ \
    readline

# Copy TDLib libraries
COPY --from=tdlib-builder /usr/local/lib/libtdjson.so* /usr/local/lib/

# Copy the built application
COPY --from=go-builder /build/tdclient /usr/local/bin/tdclient

# Create directories for data
RUN mkdir -p /app/data/tdlib /app/data/files /app/data

WORKDIR /app

# Set environment variables
ENV LD_LIBRARY_PATH=/usr/local/lib

# Volume for persistent data
VOLUME ["/app/data"]

# Run the application
ENTRYPOINT ["tdclient"]
CMD ["-config", "/app/config.yaml"]
