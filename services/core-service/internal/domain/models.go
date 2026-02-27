package domain

import (
	"encoding/json"
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

type AccountType string

const (
	AccountTypeSandbox AccountType = "sandbox"
	AccountTypeCompany AccountType = "company"
)

type SandboxAccount struct {
	ID             int64
	TypeID         string
	OwnerAccountID string
	AccountID      string
	Name           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ListSandboxAccountsResult struct {
	Sandboxes []*SandboxAccount
	PageInfo  pagination.PageInfo
}

type AccountContext struct {
	AccountID          string
	IsSandbox          bool
	OwnerAccountID     *string
	AccountMode        constants.AccountMode
	SubscriptionStatus *string
}

type AccountUser struct {
	ID           string
	UserID       string
	DepartmentID *string
	RoleID       *string
	RoleTypeCode *string
	AccountID    string
	LastUsedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AccountUserAccess struct {
	AccountUserID string
	AccountID     string
	RoleID        *string
	RoleTypeCode  *string
	Permissions   map[string]bool
	LastUsedAt    *time.Time
}

type AccountRelation struct {
	ID                    string
	CounterpartyAccountID string
	RoleCode              string
}

type AccountAffiliation struct {
	AccountID   string
	AccountName string
	RoleID      string
	RoleName    string
	LastUsedAt  *time.Time
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
	RecoveryPoint  string
}

func (k *IdempotencyKey) HasResponse() bool {
	return k.ResponseCode != nil
}

func (k *IdempotencyKey) IsFinished() bool {
	return k.RecoveryPoint == string(RecoveryPointFinished)
}
