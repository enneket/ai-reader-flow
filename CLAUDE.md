# CLAUDE.md

## Project

AI RSS Reader — Go REST API + React frontend, SQLite storage, AI-powered filtering.

## Tech Stack

- Go 1.21+, Chi router (stdlib http.ServeMux)
- React + TypeScript, Vite
- SQLite (WAL mode, connection pool: MaxOpenConns=25, MaxIdleConns=5)
- Ollama / OpenAI / Claude for AI features

## Key Commands

```bash
make dev:go      # Run API server
make dev:frontend # Run frontend dev
make build       # Build binary
make up          # Docker compose up
make down        # Docker compose down
```

## Architecture Notes

- Services are global vars in `cmd/server/main.go` (simple DI, no DI framework)
- Repository uses global `sqlite.DB` connection pool
- All list queries use limit/offset pagination
- Errors are logged via `log.Printf`, not silently ignored
- Frontend state managed via `/api/app-state` (Wails pattern, but now REST)
