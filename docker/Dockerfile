# Use official Go image for the backend build
FROM golang:1.22 AS backend-builder

# Install runtime dependencies (libvips)
RUN apt-get update && apt-get install -y --no-install-recommends \
    libvips-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Set the environment variables for the build
ENV GOOS=linux
ENV GOARCH=amd64

# Copy the source code and build the backend
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN make install-tools && make generate
RUN go build -o discuit .

# Use official Bun image for the frontend build
FROM oven/bun AS frontend-builder

# Copy required configuration files and build files
WORKDIR /app
COPY config.default.yaml .
RUN mv config.default.yaml config.yaml
COPY --from=backend-builder /app/discuit /app/discuit

# Copy the source code and build the frontend
WORKDIR /app/ui
COPY ui/package.json ui/bun.lockb ./
RUN bun install --frozen-lockfile
COPY ui/ .

# Final stage: setup the runtime environment
FROM node:18

# Avoid prompts from the package manager during build
ENV DEBIAN_FRONTEND=noninteractive

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    libc6 \
    libvips-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install bun
RUN npm install -g bun

# Copy built artifacts from previous stages
COPY --from=frontend-builder /app/ui /app/ui
COPY --from=backend-builder /app/discuit /app/discuit
COPY --from=backend-builder /app/docs/swagger.json /app/docs/swagger.json
COPY config.default.yaml /app/config.yaml
COPY migrations /app/migrations
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

WORKDIR /app

# Setup the environment and ports
EXPOSE 80
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/app/discuit", "serve"]
