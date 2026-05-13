package domain

// RepoFactory constructs repository implementations for a single database session (typically *sqlc.Queries).
type RepoFactory interface {
	NewEmailLogRepo() EmailLogRepo
	NewIdempotencyKeyRepo() IdempotencyKeyRepo
}
