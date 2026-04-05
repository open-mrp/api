package domain

import "context"

// CoreClient provides access to core-service via gRPC.
type CoreClient interface {
	SearchProducts(ctx context.Context, accountID, query string) ([]ProductResult, error)
	ListProducts(ctx context.Context, accountID string) ([]ProductResult, error)
	GetCustomerByEmail(ctx context.Context, ownerAccountID, email string) (*CustomerResult, error)
	GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, error)
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, error)
}

// AccountContext holds billing-relevant metadata for an account.
type AccountContext struct {
	IsSandbox                    bool
	OwnerAccountID               string
	PlanCode                     string
	AgentMonthlySpendingCapCents *int64
}

// BillingCustomerResolver resolves the Stripe customer ID for an account.
type BillingCustomerResolver interface {
	GetStripeCustomerID(ctx context.Context, accountID string) (string, error)
}
