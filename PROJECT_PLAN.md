# Project Plan — WhatsApp AI Food & Nutrition Tracker

Legend: `[x]` done · `[~]` in progress · `[ ]` pending

## Phase 1 — Foundation
[x] Inspect repository / environment
[x] Initialize Go project (module: github.com/yourorg/food-tracker)
[x] Basic directory structure (cmd/, internal/*, migrations/, tests/)
[x] Configuration package (env-based, with defaults + validation)
[x] PostgreSQL connection pool + health check
[x] Redis client + health check
[x] HTTP server with `/health` endpoint (checks both dependencies)
[x] Structured JSON logging + request-ID middleware
[x] Graceful shutdown (SIGINT/SIGTERM, context timeouts)
[x] Docker Compose (app + postgres + redis, with healthchecks)
[x] Multi-stage Dockerfile
[x] .env.example / .gitignore
[x] README.md
[x] Unit tests for config and HTTP layer (`go test ./...` passing)
[x] `go vet ./...` and `gofmt` clean

## Phase 2 — WhatsApp Integration
[ ] WhatsApp configuration (already scaffolded in config package)
[ ] Webhook verification (GET /webhook/whatsapp)
[ ] Incoming webhook handler (POST /webhook/whatsapp)
[ ] Message parsing (extract sender, message ID, text body)
[ ] Outgoing message sender (WhatsApp Cloud API client)
[ ] Redis-based webhook idempotency (dedupe by message ID)
[ ] Tests (mocked WhatsApp API)

## Phase 3 — Food Understanding (LLM)
[ ] LLM client (structured JSON output)
[ ] Prompt design (English + Hinglish, household measurements)
[ ] Structured output schema + validation (ranges, units, confidence)
[ ] Food parser service
[ ] Clarification-question detection
[ ] Mocked tests (no real API key required)

## Phase 4 — Nutrition Provider
[ ] Evaluate nutrition data sources (USDA FoodData Central, Open Food Facts, etc.)
[ ] Document the chosen provider and rationale
[ ] NutritionProvider interface
[ ] API client implementation
[ ] Nutrition data normalization
[ ] Redis caching of nutrition lookups
[ ] Tests (mocked provider)

## Phase 5 — Calculation Engine
[ ] FoodItem → NutritionData → CalculationEngine → NutritionSummary
[ ] Deterministic per-100g scaling + aggregation
[ ] Unit tests: single item, multiple items, decimals, zero/invalid values,
    large quantities, missing nutrition data

## Phase 6 — Conversation Manager
[ ] Redis-backed conversation state (multi-turn clarification flows)
[ ] State transitions (awaiting clarification → resolved → logged)
[ ] Persist completed food logs to PostgreSQL
[ ] Tests

## Phase 7 — End-to-End Integration
[ ] Wire WhatsApp → Conversation Manager → LLM → Nutrition → Calculation → Postgres → WhatsApp
[ ] End-to-end tests with mocked external services

## Phase 8 — Hardening
[ ] Race condition review
[ ] Duplicate-processing review
[ ] Timeouts/retries on all external calls
[ ] Input validation audit
[ ] Security review (secrets, logging of sensitive data)
[ ] Database index review
[ ] Redis key expiration review
[ ] Context cancellation / graceful shutdown review

## Phase 9 — Deployment
[ ] Docker build documentation
[ ] Environment variable reference
[ ] Managed Postgres/Redis recommendations (low-cost)
[ ] HTTPS / webhook exposure guidance
[ ] Secrets management guidance
[ ] Health checks in deployment config
[ ] Logging/observability in production

---