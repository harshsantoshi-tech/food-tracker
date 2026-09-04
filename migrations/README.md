# Migrations

This project uses [golang-migrate](https://github.com/golang-migrate/migrate)
for schema migrations (SQL files, no ORM magic, easy to review in PRs).

No schema migrations exist yet — the `users`, `food_logs`, `food_items`,
and `conversation_state` tables will be introduced starting in the phase
that first needs persistence (WhatsApp user identification / food logging).

## Conventions (once migrations are added)

- Files are named `NNNN_description.up.sql` / `NNNN_description.down.sql`.
- Every table has `id`, `created_at`, and (where mutable) `updated_at`.
- Every migration must have a working `down` migration.
- Foreign keys use `ON DELETE` behavior appropriate to the relationship.

## Running migrations (once added)

\`\`\`bash
migrate -path migrations -database "$POSTGRES_DSN" up
migrate -path migrations -database "$POSTGRES_DSN" down 1
\`\`\`