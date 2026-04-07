# Docker 容器整合设计

## 目标

将 `api`（Go server）和 `web`（Nginx + React 静态文件）两个容器合并为一个，简化部署、减少资源开销。

## 当前架构

```
docker-compose
├── api (Go binary on port 5562)
└── web (Nginx + React static on port 80)
```

## 目标架构

```
单一容器
├── supervisord (进程管理)
│   ├── nginx (port 80, serves static + proxies /api/*)
│   └── ./server (Go API on localhost:5562)
└── /app/
    ├── dist/ (React 构建产物)
    ├── nginx.conf
    └── server (Go binary)
```

## 实现方案

### Dockerfile

Multi-stage build：
1. **Builder stage** — Node.js build React → `/app/dist`
2. **Go stage** — Compile Go → `/app/server`
3. **Final stage** — Debian base
   - COPY `/app/dist` → nginx html dir
   - COPY Go binary + nginx.conf
   - Install `supervisord`
   - ENTRYPOINT 启动 supervisord

### 关键配置

**supervisord.conf：**
```
[supervisord]
nodaemon=true

[program:nginx]
command=nginx -g "daemon off;"
stdout_logfile=/dev/stdout
stderr_logfile=/dev/stderr

[program:server]
command=/app/server
env vars: DB_PATH, PORT, HTTP_PROXY, etc.
stdout_logfile=/dev/stdout
stderr_logfile=/dev/stderr
```

**nginx.conf：**
```
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    location /api/ {
        proxy_pass http://localhost:5562;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

### docker-compose 变更

- 移除 `web` service
- `api` service 改为同时 build 和托管静态文件
- port 改为 `8080:80`（单端口暴露）
- 环境变量保持不变

## 验证步骤

1. `docker compose up --build`
2. 访问 `localhost:8080` → React UI
3. API 调用 `localhost:8080/api/*` → Go responses
4. 刷新订阅源功能正常
5. 生成简报功能正常

## 文件变更

- 新建 `Dockerfile`（合并原 `Dockerfile.api` + `Dockerfile.web`）
- 新建 `supervisord.conf`
- 更新 `nginx.conf`（增加 proxy 配置）
- 更新 `docker-compose.yml`（单 service）
- 删除 `Dockerfile.api`、`Dockerfile.web`

## 不变更内容

- Go API 代码不变
- React 前端代码不变
- 环境变量配置方式不变
