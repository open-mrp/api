package domain

// RepoFactory provides repository instances that share the same underlying
// sqlc query executor (plain DB access or a transaction).
type RepoFactory interface {
	NewUserRepo() UserRepo
	NewRefreshTokenRepo() RefreshTokenRepo
	NewAccountUserRepo() AccountUserRepo
	NewAccountRelationRepo() AccountRelationRepo
	NewAPIKeyRepo() APIKeyRepo
	NewRolePermissionRepo() RolePermissionRepo
}

// Mediators groups all mediator dependencies built for a specific repository factory.
type Mediators struct {
	User         UserMed
	APIKey       APIKeyMed
	AccountUser  AccountUserMed
	Password     PasswordMed
	RefreshToken RefreshTokenMed
}

// MediatorFactory builds mediators bound to a given repository factory (e.g., per transaction).
type MediatorFactory interface {
	Build(RepoFactory) Mediators
}
