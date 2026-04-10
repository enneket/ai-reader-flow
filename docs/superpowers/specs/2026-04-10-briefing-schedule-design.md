# Design: 多时间点简报定时配置

## Problem Statement

当前 cron 只支持单一时间点（每天 09:00），无法配置多个执行时间（如早9点、下午6点、晚9点各一次）。

## Solution

支持配置多个执行时间，时间列表存储在 `config.toml`，支持运行时动态读取和更新。

## Architecture

- **单一 cron** — 每分钟触发一次（`@hourly`），cron handler 内检查当前时间是否在列表中
- **动态生效** — 时间列表在配置文件 (`config.toml`)，增删无需重启服务
- **统一行为** — 匹配到时间点 → `RefreshAllFeeds` → `GenerateBriefingWithProgress`

## Config Change

```toml
[Cron]
Enabled = true
Times = ["09:00", "18:00", "21:00"]   # 北京时间，多个时间点
```

废弃字段：`Hour`（单个小时）、`Minute`（单个分钟）、`IntervalMins`（间隔分钟）。

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/cron-times` | 获取所有定时时间 |
| PUT | `/api/cron-times` | 批量更新定时时间 |

### GET /api/cron-times
Response: `["09:00", "18:00", "21:00"]`

### PUT /api/cron-times
Request: `{"times": ["09:00", "18:00"]}`
Response: `{"success": true}`

## Data Flow

1. Config loader reads `Times []string` from `config.toml`
2. Cron fires every minute (`"* * * * *"`)
3. Cron handler: `now.Format("15:04")` vs `config.Cron.Times`
4. If match: run `RefreshAllFeeds` → `GenerateBriefingWithProgress`
5. If no match: skip silently

## Frontend

Settings 页面新增「简报定时」区块：

- 时间列表展示（HH:MM 格式）
- 输入框 + 添加按钮（校验格式，阻止重复）
- 每项右侧删除按钮
- 保存即时生效（调用 PUT API）

## Files to Change

### Backend
- `internal/config/config.go` — `CronConfig` 新增 `Times []string`，废弃 `Hour`/`Minute`/`IntervalMins`
- `cmd/server/main.go` — cron 改 `@hourly`，handler 内做时间列表匹配；新增 `/api/cron-times` handlers

### Frontend
- `frontend/src/api.ts` — 新增 `getCronTimes`, `setCronTimes`
- `frontend/src/components/Settings.tsx` — 新增简报定时 UI

## Testing

- 单元测试：时间匹配逻辑
- 手动测试：修改配置后观察 cron 是否按新时间执行

## Open Questions

无
