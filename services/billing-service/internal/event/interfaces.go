package event

import (
	"context"
	"time"

	apierror "github.com/augno/api/shared/errors"
)

// WebhookCoreClient defines the core-service operations needed by the webhook consumer.
type WebhookCoreClient interface {
	GetAccountByStripeCustomerID(ctx context.Context, stripeCustomerID string) (string, string, *apierror.APIError)
	UpdateAccountSubscription(ctx context.Context, idempotencyKey, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string, billingProfileID *string, billingCadenceID *string, pricingPlanSubscriptionID *string, servicingStatus *string, collectionStatus *string) *apierror.APIError
	ClearAccountStripeCustomer(ctx context.Context, idempotencyKey, accountID string) *apierror.APIError
}

// EventLogRepo defines the event log operations needed by the webhook consumer.
type EventLogRepo interface {
	Exists(ctx context.Context, eventID, objectID string) (bool, error)
	Insert(ctx context.Context, eventID, eventType, objectID string) error
}

// WebhookNotificationClient defines the notification operations needed by the webhook consumer.
type WebhookNotificationClient interface {
	SendPaymentActionRequired(ctx context.Context, accountID, adminEmail string) *apierror.APIError
}

// WebhookAccountUsageRepo defines the account lookup operations needed by the webhook consumer.
type WebhookAccountUsageRepo interface {
	GetAdminEmailByAccountID(ctx context.Context, accountID string) (string, *apierror.APIError)
}
