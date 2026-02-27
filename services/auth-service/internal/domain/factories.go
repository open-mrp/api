package domain

import "github.com/augno/api/shared/messaging"

// RepoFactory provides repository instances that share the same underlying
// sqlc query executor (plain DB access or a transaction).
type RepoFactory interface {
	NewUserRepo() UserRepo
	NewRefreshTokenRepo() RefreshTokenRepo
	NewAPIKeyRepo() APIKeyRepo
	NewDocAPIKeyRepo() DocAPIKeyRepo
	NewRegistrationSessionRepo() RegistrationSessionRepo
	NewRegistrationQueueRepo() RegistrationQueueRepo
	NewIdempotencyKeyRepo() IdempotencyKeyRepo
	NewOutboxRepo() messaging.OutboxRepo
}

// Mediators groups all mediator dependencies built for a specific repository factory.
type Mediators struct {
	User         UserMed
	APIKey       APIKeyMed
	DocAPIKey    DocAPIKeyMed
	Password     PasswordMed
	RefreshToken RefreshTokenMed
	Idempotency  IdempotencyMed
	Registration RegistrationMed
}

// MediatorFactory builds mediators bound to a given repository factory (e.g., per transaction).
type MediatorFactory interface {
	Build(RepoFactory) Mediators
}
