# Stage 1: Build React
FROM node:20-alpine AS frontend_builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go
FROM golang:bookworm AS go_builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

# Final stage
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    nginx \
    supervisor \
    && rm -rf /var/lib/apt/lists/*

# Setup supervisord
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Copy Go binary
COPY --from=go_builder /src/server /app/server

# Copy React static files
COPY --from=frontend_builder /app/dist /usr/share/nginx/html

# Copy nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
