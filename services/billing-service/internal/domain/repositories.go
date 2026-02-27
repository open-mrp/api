package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
)

type PricingPlanRepo interface {
	GetPlanByCode(ctx context.Context, planCode string) (*PricingPlan, *apierror.APIError)
	GetPlanByTypeID(ctx context.Context, typeID string) (*PricingPlan, *apierror.APIError)
	GetPlanLimitsByTypeID(ctx context.Context, typeID string) ([]PlanLimit, *apierror.APIError)
	ListPricingPlans(ctx context.Context, cursor *string, limit int32) ([]PricingPlan, pagination.PageInfo, *apierror.APIError)
}

type AccountUsageRepo interface {
	GetLimitsByAccountID(ctx context.Context, accountID string) ([]PlanLimit, *apierror.APIError)
	CountUsersByAccountID(ctx context.Context, accountID string) (int, *apierror.APIError)
	CountSandboxesByAccountID(ctx context.Context, accountID string) (int, *apierror.APIError)
	CountInvoicesByAccountID(ctx context.Context, accountID string) (int, *apierror.APIError)
	CountBatchesByAccountID(ctx context.Context, accountID string) (int, *apierror.APIError)
	GetAccountSubscriptionInfo(ctx context.Context, accountID string) (*AccountSubscriptionInfo, *apierror.APIError)
	GetStripeCustomerIDByAccountID(ctx context.Context, accountID string) (*string, *apierror.APIError)
	GetAccountNameAndPlanCode(ctx context.Context, accountID string) (name string, planCode string, apiErr *apierror.APIError)
	GetUserEmailByID(ctx context.Context, userID string) (email string, displayName *string, apiErr *apierror.APIError)
	GetAdminEmailByAccountID(ctx context.Context, accountID string) (string, *apierror.APIError)
	UpdateStripeCustomerIDByAccountID(ctx context.Context, stripeCustomerID, accountID string) *apierror.APIError
}
