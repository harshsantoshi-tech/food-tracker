package storage

import "context"

// UserIDResolver adapts UserRepository to the narrow shape the
// whatsapp package needs (just a phone number → user ID lookup),
// without whatsapp needing to know anything about the full User model
// or how it's persisted.
type UserIDResolver struct {
	repo *UserRepository
}

func NewUserIDResolver(repo *UserRepository) *UserIDResolver {
	return &UserIDResolver{repo: repo}
}

func (r *UserIDResolver) ResolveUserID(ctx context.Context, whatsappNumber string) (int64, error) {
	user, err := r.repo.GetOrCreateByWhatsAppNumber(ctx, whatsappNumber)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}