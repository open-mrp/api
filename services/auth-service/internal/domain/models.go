package domain

import (
	"encoding/json"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
)

type AuthAccountRelation struct {
	ID                      string
	CounterpartyAccountID   string
	AccountRelationRoleCode types.IdentityRelationType
	IsOwnerSide             bool
}

type RefreshToken struct {
	Token     string
	UserID    string     `audit:"user_id"`
	ExpiresAt time.Time  `audit:"expires_at"`
	RevokedAt *time.Time `audit:"revoked_at"`
}

// IsRevoked reports whether the refresh token has been revoked.
func (m *RefreshToken) IsRevoked() bool {
	return m.RevokedAt != nil
}

// IsExpired reports whether the refresh token has passed its expiration time.
func (m *RefreshToken) IsExpired() bool {
	return m.ExpiresAt.Before(time.Now().UTC())
}

type IdempotencyKey struct {
	ID             int64
	TypeID         string
	ServiceName    string
	Handler        string
	IdempotencyKey string
	ActorID        *string
	IdentityType   string
	ScopeHash      string
	ResponseCode   *int
	ResponseBody   json.RawMessage
	RecoveryPoint  RecoveryPoint
}

func (k *IdempotencyKey) HasResponse() bool {
	return k.ResponseCode != nil
}

func (k *IdempotencyKey) IsFinished() bool {
	return k.RecoveryPoint.IsFinished()
}
