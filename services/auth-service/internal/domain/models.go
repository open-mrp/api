package domain

import (
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
)

type AccountAffiliation struct {
	AccountID   string
	AccountName string
	RoleID      string
	RoleName    string
	LastUsedAt  *time.Time
}

type AccountUser struct {
	ID           string
	Name         *string
	UserID       string
	DepartmentID *string
	RoleID       *string
	RoleTypeCode *string
	AccountID    string
	LastUsedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type APIKey struct {
	ID             string
	Name           string
	LastFour       string
	SecretHash     []byte
	OwnerAccountID string
	RoleID         string
	RoleTypeCode   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastUsedAt     *time.Time
	ExpiresAt      *time.Time
	RevokedAt      *time.Time
}

type AuthAccountRelation struct {
	ID                      string
	CounterpartyAccountID   string
	AccountRelationRoleCode types.IdentityActorType
}

type RefreshToken struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type RolePermission struct {
	ID             string
	Create         bool
	Read           bool
	Update         bool
	Delete         bool
	RoleID         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	PermissionCode string
}

type ParsedAPIKey struct {
	AccountMode constants.AccountMode
	ID          string
	Secret      string
	Checksum    string
}

func (p *ParsedAPIKey) String() string {
	return string(types.APIKeyPrefixSecretKey) + string(p.AccountMode) + "_" + p.ID + "_" + p.Secret + p.Checksum
}
