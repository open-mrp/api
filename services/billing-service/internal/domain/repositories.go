package domain

import (
	"context"
	"encoding/json"
	"time"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
)

type PricingPlanRepo interface {
	GetPlanByCode(ctx context.Context, planCode string) (*PricingPlan, *apierror.APIError)
	GetPlanByTypeID(ctx context.Context, typeID string) (*PricingPlan, *apierror.APIError)
	GetPlanLimitsByTypeID(ctx context.Context, typeID string) ([]PlanLimit, *apierror.APIError)
	ListPricingPlans(ctx context.Context, cursor *string, limit int32, query *string) ([]PricingPlan, pagination.PageInfo, *apierror.APIError)
}

type AgentTokenBillingRepo interface {
	UpsertAgentTokenBilling(ctx context.Context, params UpsertAgentTokenBillingParams) *apierror.APIError
	GetByAccountAndPeriod(ctx context.Context, accountID string, periodStart time.Time) (*AgentTokenBilling, *apierror.APIError)
	GetUsageSummary(ctx context.Context, accountID string, periodStart time.Time) (int64, *apierror.APIError)
	GetCompletedTokensByAccount(ctx context.Context, accountID string) (int64, *apierror.APIError)
}

type UpsertAgentTokenBillingParams struct {
	ID           string
	AccountID    string
	PeriodStart  time.Time
	PeriodEnd    time.Time
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
}

type AccountUsageRepo interface {
	GetLimitsByAccountID(ctx context.Context, accountID string) ([]PlanLimit, *apierror.APIError)
	CountUsersByAccountID(ctx context.Context, accountID string) (int, *apierror.APIError)
	CountSandboxesByAccountID(ctx context.Context, accountID string) (int, *apierror.APIError)
	CountInvoicesByAccountID(ctx context.Context, accountID string, periodStart time.Time) (int, *apierror.APIError)
	CountBatchesByAccountID(ctx context.Context, accountID string, periodStart time.Time) (int, *apierror.APIError)
	GetAccountSubscriptionInfo(ctx context.Context, accountID string) (*AccountSubscriptionInfo, *apierror.APIError)
	GetStripeCustomerIDByAccountID(ctx context.Context, accountID string) (*string, *apierror.APIError)
	GetAccountNameAndPlanCode(ctx context.Context, accountID string) (name string, planCode string, apiErr *apierror.APIError)
	GetUserEmailByID(ctx context.Context, userID string) (email string, displayName *string, apiErr *apierror.APIError)
	GetAdminEmailByAccountID(ctx context.Context, accountID string) (string, *apierror.APIError)
	UpdateStripeCustomerIDByAccountID(ctx context.Context, stripeCustomerID, accountID string) *apierror.APIError
}

type IdempotencyKeyRepo interface {
	GetByScopeHash(ctx context.Context, scopeHash string) (*IdempotencyKey, *apierror.APIError)
	Create(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, *apierror.APIError)
	AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint RecoveryPoint) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, typeID string) (RecoveryPoint, *apierror.APIError)
	SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint RecoveryPoint) *apierror.APIError
}
