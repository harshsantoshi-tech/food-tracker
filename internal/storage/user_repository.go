package storage

import (
	"context"
	"fmt"
)

// UserRepository manages persistence of users.
type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetOrCreateByWhatsAppNumber finds an existing user by their WhatsApp
// number, or creates one if this is their first message. Using
// "INSERT ... ON CONFLICT DO UPDATE" makes this atomic — no separate
// SELECT-then-INSERT race condition if two messages from a brand new
// number arrive concurrently.
func (r *UserRepository) GetOrCreateByWhatsAppNumber(ctx context.Context, whatsappNumber string) (*User, error) {
	const query = `
		INSERT INTO users (whatsapp_number)
		VALUES ($1)
		ON CONFLICT (whatsapp_number) DO UPDATE SET updated_at = now()
		RETURNING id, whatsapp_number, name, created_at, updated_at
	`

	var u User
	var name *string
	err := r.db.QueryRowContext(ctx, query, whatsappNumber).Scan(
		&u.ID, &u.WhatsAppNumber, &name, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get or create user %s: %w", whatsappNumber, err)
	}
	if name != nil {
		u.Name = *name
	}
	return &u, nil
}