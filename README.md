# AI RSS Reader

AI-powered RSS reader with semantic deduplication, quality scoring, and automated news briefing generation.

## Features

- **RSS Aggregation** — Subscribe to RSS feeds, automatic refresh via cron
- **AI Filtering** — Semantic deduplication and quality scoring powered by Ollama/OpenAI/Claude
- **Smart Reading** — Filter by acceptance status, save articles as notes
- **News Briefing** — Automated daily briefing that clusters articles by topic, generates summaries
- **Multi-language** — Chinese/English UI

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   React     │────▶│   Go API     │────▶│   SQLite    │
│   Frontend  │     │   :8080      │     │   WAL mode   │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  Ollama /   │
                    │  OpenAI /   │
                    │  Claude     │
                    └─────────────┘
```

- **Backend**: Go REST API (`cmd/server/main.go`)
- **Frontend**: React + TypeScript + Vite (`frontend/`)
- **Database**: SQLite with WAL mode (MaxOpenConns=25, MaxIdleConns=5)
- **AI**: Ollama (default), OpenAI, or Claude via `/api/ai-config`
- **Deployment**: Docker + Nginx, single container with supervisord

## Quick Start

```bash
# Start all services
docker compose up -d

# Dev mode
make dev:go        # API server on :8080
make dev:frontend  # Vite dev server on :5173

# Build
make build
```

## Routes

### Feeds

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/feeds` | List all feeds |
| POST | `/api/feeds` | Add RSS feed |
| DELETE | `/api/feeds/{id}` | Delete feed |
| POST | `/api/feeds/{id}/refresh` | Refresh single feed |
| POST | `/api/refresh` | Refresh all feeds (cron or manual) |

### Articles

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/articles` | List articles |
| GET | `/api/articles/{id}` | Get article detail |
| POST | `/api/articles/{id}/accept` | Mark accepted |
| POST | `/api/articles/{id}/reject` | Mark rejected |
| POST | `/api/articles/{id}/snooze` | Snooze article |
| POST | `/api/articles/{id}/summary` | Generate AI summary |
| POST | `/api/articles/{id}/note` | Save as note |

### Briefing

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/briefings` | List briefings (paginated) |
| GET | `/api/briefings/{id}` | Get briefing detail |
| POST | `/api/briefing/generate` | Generate new briefing |

### Notes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/notes` | List notes |
| DELETE | `/api/notes/{id}` | Delete note |

### Config

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/ai-config` | Get AI provider config |
| PUT | `/api/ai-config` | Update AI provider config |
| GET | `/api/app-state` | Get app UI state |
| PUT | `/api/app-state` | Update app UI state |

## Query Parameters

**Articles**: `?feedId=&filterMode=&limit=&offset=`

**Briefings**: `?limit=&offset=`

**Filter Modes**: `all` | `filtered` | `saved` | `unread` | `accepted` | `rejected` | `snoozed`

## Briefing

The briefing feature generates an automated news briefing by:

1. Fetching latest articles from all feeds
2. Grouping articles by topic using AI
3. Generating concise summaries for each topic
4. Presenting as a structured report with source links

Briefings are titled by timestamp (e.g., `09时00分42秒 简报`) and support real-time generation status polling.

## License

MIT
