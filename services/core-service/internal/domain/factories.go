package domain

import "github.com/augno/api/shared/messaging"

type RepoFactory interface {
	NewAccountRepo() AccountRepo
	NewAccountUserRepo() AccountUserRepo
	NewAccountRelationRepo() AccountRelationRepo
	NewRolePermissionRepo() RolePermissionRepo
	NewSandboxAccountRepo() SandboxAccountRepo
	NewRegistrationRepo() RegistrationRepo
	NewIdempotencyKeyRepo() IdempotencyKeyRepo
	NewUnitRepo() UnitRepo
	NewOutboxRepo() messaging.OutboxRepo
}

// Mediators groups all mediator dependencies built for a specific repository factory.
type Mediators struct {
	Sandbox     SandboxMed
	Idempotency IdempotencyMed
}

// MediatorFactory builds mediators bound to a given repository factory (e.g., per transaction).
type MediatorFactory interface {
	Build(RepoFactory) Mediators
}
