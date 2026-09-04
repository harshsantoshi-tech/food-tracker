# Food Tracker — WhatsApp AI Nutrition Bot

A WhatsApp-based food tracking service. A user texts what they ate in
plain language (English or Hinglish); the system identifies the food
items and portions using an LLM, looks up real nutrition data, and
replies with an estimated calorie/protein/carb/fat breakdown.

## Architecture

\`\`\`
WhatsApp
   ↓
WhatsApp Cloud API
   ↓
Go Backend (this repo)
   ↓
LLM Food Parser        — understands language, extracts structured food items
   ↓
Structured Food Items  — validated JSON: name, quantity, unit, weight, confidence
   ↓
Nutrition Provider      — looks up nutrition-per-100g from a real database/API
   ↓
Calculation Engine      — deterministic: nutrition_per_100g × grams / 100
   ↓
Nutrition Result
   ↓
WhatsApp Response
\`\`\`

**Key principle:** the LLM never invents final calorie/macro numbers. It
only converts natural language into structured food + portion data. All
nutrition math is done deterministically in Go against real nutrition
data, so results are reproducible and auditable.

## Project status

This repo is being built incrementally, phase by phase. See
[`PROJECT_PLAN.md`](./PROJECT_PLAN.md) for what's done and what's next.

**Phase 1 (current):** foundation only — Go service skeleton, PostgreSQL,
Redis, Docker Compose, configuration, and a `/health` endpoint. WhatsApp,
the LLM parser, and the nutrition provider are not wired up yet.

## Project layout

\`\`\`
food-tracker/
├── cmd/server/          # main.go — entrypoint, wiring, graceful shutdown
├── internal/
│   ├── config/           # env-based configuration loading
│   ├── http/             # HTTP server, routing, middleware, health checks
│   ├── whatsapp/         # WhatsApp Cloud API client + webhook (Phase 2)
│   ├── llm/               # LLM food parser (Phase 3)
│   ├── food/              # food parsing/validation types (Phase 3)
│   ├── nutrition/         # nutrition provider interface + implementation (Phase 4)
│   ├── calculation/       # deterministic nutrition math (Phase 5)
│   ├── conversation/      # multi-turn conversation state (Phase 6)
│   ├── storage/           # PostgreSQL connection + repositories
│   └── cache/             # Redis connection + helpers
├── migrations/            # SQL schema migrations (golang-migrate)
├── tests/                 # end-to-end tests (added in later phases)
├── docker-compose.yml
├── Dockerfile
└── .env.example
\`\`\`

## Local development

### Prerequisites

- Go 1.22+
- Docker + Docker Compose (recommended), OR local PostgreSQL 16 + Redis 7

### Option A — Docker Compose (recommended)

\`\`\`bash
cp .env.example .env
docker compose up --build
\`\`\`

This starts the app, PostgreSQL, and Redis together. The app waits for
both dependencies to report healthy before starting.

### Option B — Run natively against local Postgres/Redis

\`\`\`bash
cp .env.example .env
# edit .env if your local Postgres/Redis differ from the defaults

go run ./cmd/server
\`\`\`

### Health check

\`\`\`bash
curl http://localhost:8080/health
\`\`\`

\`\`\`json
{
  "status": "ok",
  "components": {
    "postgres": { "status": "ok" },
    "redis": { "status": "ok" }
  }
}
\`\`\`

### Running tests

\`\`\`bash
go test ./...
go vet ./...
gofmt -l .
\`\`\`

## Configuration

All configuration is via environment variables — see
[`.env.example`](./.env.example) for the full list. Never commit a real
`.env` file; secrets (WhatsApp tokens, LLM API keys, DB passwords) must
never be hardcoded.

## Design notes

- **Separation of concerns:** the LLM only extracts structured food data;
  a separate deterministic calculation engine does all the arithmetic.
  This keeps nutrition numbers reproducible and makes the system testable
  without hitting a real LLM.
- **PostgreSQL is the source of truth** for users, food logs, and food
  items. Redis is used only for ephemeral data: conversation state,
  webhook-dedup keys, nutrition response caching, and rate limiting.
- **Idempotency:** incoming WhatsApp webhook messages are deduplicated by
  message ID before processing (Phase 2).