package domain

import (
	"context"
	"encoding/json"

	apierror "github.com/augno/api/shared/errors"
)

type EmailLogRepo interface {
	Create(ctx context.Context, emailLog *EmailLog) *apierror.APIError
	FindBySesMessageID(ctx context.Context, sesMessageID string) (*EmailLog, *apierror.APIError)
}

// IdempotencyKeyRepo persists idempotency rows for handlers that opt into contracts idempotency.
// Instances are constructed via RepoFactory.NewIdempotencyKeyRepo(); no transport layer wires it yet.
type IdempotencyKeyRepo interface {
	GetByScopeHash(ctx context.Context, scopeHash string) (*IdempotencyKey, *apierror.APIError)
	Create(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, *apierror.APIError)
	AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint RecoveryPoint) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, typeID string) (string, *apierror.APIError)
	SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint RecoveryPoint) *apierror.APIError
}
