# Docker 容器整合实现方案

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 api 和 web 两个容器合并为一个，用 supervisord 同时管理 nginx 和 Go 进程。

**Architecture:** 单 Dockerfile multi-stage build，supervisord 管理 nginx(80) + Go(5562) 两个进程，nginx 代理 /api/* 到 localhost:5562，同时托管 React 静态文件。

**Tech Stack:** Go, React/Vite, nginx, supervisord, debian-bookworm-slim

---

## 文件变更概览

| 文件 | 操作 |
|------|------|
| `Dockerfile` (new) | 合并 build Go + React，统一镜像 |
| `supervisord.conf` (new) | 进程管理配置 |
| `nginx.conf` | 修改 proxy 指向 localhost:5562 |
| `docker-compose.yml` | 合并为单 service |
| `Dockerfile.api` | 删除 |
| `Dockerfile.web` | 删除 |

---

## Task 1: 创建统一 Dockerfile

**Files:**
- Create: `Dockerfile`
- Delete: `Dockerfile.api`, `Dockerfile.web`

```dockerfile
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
```

---

## Task 2: 创建 supervisord.conf

**Files:**
- Create: `supervisord.conf`

```ini
[supervisord]
nodaemon=true
user=root
logfile=/dev/null
logfile_maxbytes=0
pidfile=/var/run/supervisord.pid
loglevel=info

[program:nginx]
command=nginx -g "daemon off;"
stdout_logfile=/dev/stdout
stderr_logfile=/dev/stderr
autostart=true
autorestart=true
priority=10

[program:server]
command=/app/server
environment=DB_PATH="/data/reader.db",PORT="5562"
stdout_logfile=/dev/stdout
stderr_logfile=/dev/stderr
autostart=true
autorestart=true
priority=20
```

---

## Task 3: 更新 nginx.conf

**Files:**
- Modify: `nginx.conf` (change `api:5562` → `localhost:5562`)

```nginx
upstream api_backend {
    server localhost:5562;
}

server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    # API proxy
    location /api/ {
        proxy_pass http://api_backend/api/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Connection '';
        proxy_set_header X-Accel-Buffering no;
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 86400s;
        proxy_buffering off;
        proxy_cache off;
        chunked_transfer_encoding on;
    }

    # OPML proxy
    location /opml {
        proxy_pass http://api_backend/opml;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Connection '';
    }

    # SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

---

## Task 4: 更新 docker-compose.yml

**Files:**
- Modify: `docker-compose.yml`

```yaml
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:80"
    volumes:
      - ./data:/data
    environment:
      - DB_PATH=/data/reader.db
      - PORT=5562
    extra_hosts:
      - "news.ycombinator.com:140.82.114.34"
    restart: unless-stopped

volumes:
  data:
```

---

## Task 5: 删除旧文件

**Files:**
- Delete: `Dockerfile.api`, `Dockerfile.web`

---

## Task 6: 构建验证

- [ ] `docker compose build`
- [ ] `docker compose up -d`
- [ ] `curl http://localhost:8080/` → React UI HTML
- [ ] `curl http://localhost:8080/api/progress` → `{"operation":"idle"}`
- [ ] 浏览器访问 `http://localhost:8080` 功能正常
- [ ] 刷新订阅源功能正常
- [ ] 生成简报功能正常

---

## Task 7: 提交

```bash
git add Dockerfile supervisord.conf nginx.conf docker-compose.yml
git rm Dockerfile.api Dockerfile.web
git commit -m "refactor: consolidate into single container with supervisord"
```
