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
	ID               int64
	TypeID           string
	OwnerAccountID   string
	AccountID        string
	Name             string `audit:"name"`
	OwnerAccountName *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ListSandboxAccountsResult struct {
	Sandboxes []*SandboxAccount
	PageInfo  pagination.PageInfo
}

type AccountContext struct {
	AccountID                    string
	IsSandbox                    bool
	OwnerAccountID               *string
	AccountMode                  constants.AccountMode
	SubscriptionStatus           *string
	PlanCode                     string
	AgentMonthlySpendingCapCents *int64
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
	// IsOwnerSide is true when the caller's account is the owner of the relation
	// (i.e. the API key belongs to the merchant targeting a customer/supplier account).
	IsOwnerSide bool
}

type AccountAffiliation struct {
	AccountID    string
	AccountName  string
	RoleID       string
	RoleName     string
	RoleTypeCode string
	LastUsedAt   *time.Time
}

type RoleInfo struct {
	ID           string
	Name         string
	RoleTypeCode string
}

type ProductInfo struct {
	ProductID   string
	ItemID      string
	SKU         string
	Description string
	UnitPrice   string
}

type CustomerByEmail struct {
	RelationID            string
	OwnerAccountID        string
	CounterpartyAccountID string
	RoleCode              string
	Alias                 string
	Email                 string
	UserName              string
}

// UserRecord represents a row from the user table.
type UserRecord struct {
	ID             string
	Email          *string `audit:"email"`
	Name           *string `audit:"name"`
	Username       *string `audit:"username"`
	HashedPassword *string
	EmailVerified  *time.Time `audit:"email_verified"`
	ImageURL       *string    `audit:"image_url"`
	StatusCode     string     `audit:"status_code"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateUserRecordParams are the parameters for creating a user record.
type CreateUserRecordParams struct {
	Name           *string
	Email          *string
	Username       *string
	HashedPassword *string
}

// UpdateUserParams are the parameters for updating a user record.
type UpdateUserParams struct {
	Name          *string
	ImageURL      *string
	EmailVerified *time.Time
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
