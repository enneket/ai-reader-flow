# AI RSS Reader

AI-powered RSS reader with semantic deduplication and quality scoring.

## Architecture

- **Backend**: Go REST API (`cmd/server/main.go`)
- **Frontend**: React + TypeScript (`frontend/`)
- **Database**: SQLite with WAL mode
- **Deployment**: Docker + Nginx

## Quick Start

```bash
# Start all services
docker compose up -d

# Or dev mode
make dev:go      # API server on :8080
make dev:frontend # Vite dev server on :5173
```

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/feeds | List all feeds |
| POST | /api/feeds | Add RSS feed |
| DELETE | /api/feeds/{id} | Delete feed |
| POST | /api/feeds/{id}/refresh | Refresh single feed |
| POST | /api/refresh | Refresh all feeds |
| GET | /api/articles | List articles (?feedId=&filterMode=&limit=&offset=) |
| GET | /api/articles/{id} | Get article |
| POST | /api/articles/{id}/accept | Mark accepted |
| POST | /api/articles/{id}/reject | Mark rejected |
| POST | /api/articles/{id}/snooze | Snooze |
| POST | /api/articles/{id}/summary | Generate AI summary |
| POST | /api/articles/{id}/note | Save as note |
| GET | /api/notes | List notes |
| DELETE | /api/notes/{id} | Delete note |
| GET | /api/ai-config | Get AI config |
| PUT | /api/ai-config | Update AI config |

## Filter Modes

`all` | `filtered` | `saved` | `unread` | `accepted` | `rejected` | `snoozed`

## AI Providers

Supports Ollama (default), OpenAI, and Claude. Configure via `/api/ai-config`.
