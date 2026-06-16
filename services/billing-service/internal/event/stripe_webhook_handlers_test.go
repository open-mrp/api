package event

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCoreClient struct {
	getAccountByStripeCustomerID func(ctx context.Context, stripeCustomerID string) (string, string, *apierror.APIError)
	updateAccountSubscription    func(ctx context.Context, idempotencyKey, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string, billingProfileID *string, billingCadenceID *string, pricingPlanSubscriptionID *string, servicingStatus *string, collectionStatus *string) *apierror.APIError
	clearAccountStripeCustomer   func(ctx context.Context, idempotencyKey, accountID string) *apierror.APIError
	recordOrderPayment           func(ctx context.Context, idempotencyKey, salesOrderID, paymentIntentID string) *apierror.APIError
}

func (s *stubCoreClient) GetAccountByStripeCustomerID(ctx context.Context, stripeCustomerID string) (string, string, *apierror.APIError) {
	return s.getAccountByStripeCustomerID(ctx, stripeCustomerID)
}

func (s *stubCoreClient) UpdateAccountSubscription(ctx context.Context, idempotencyKey, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string, billingProfileID *string, billingCadenceID *string, pricingPlanSubscriptionID *string, servicingStatus *string, collectionStatus *string) *apierror.APIError {
	return s.updateAccountSubscription(ctx, idempotencyKey, accountID, status, planCode, stripeSubID, periodEnd, stripeCustomerID, billingProfileID, billingCadenceID, pricingPlanSubscriptionID, servicingStatus, collectionStatus)
}

func (s *stubCoreClient) ClearAccountStripeCustomer(ctx context.Context, idempotencyKey, accountID string) *apierror.APIError {
	return s.clearAccountStripeCustomer(ctx, idempotencyKey, accountID)
}

func (s *stubCoreClient) RecordOrderPayment(ctx context.Context, idempotencyKey, salesOrderID, paymentIntentID string) *apierror.APIError {
	if s.recordOrderPayment != nil {
		return s.recordOrderPayment(ctx, idempotencyKey, salesOrderID, paymentIntentID)
	}
	return nil
}

type stubNotificationClient struct {
	sendPaymentActionRequired func(ctx context.Context, accountID, adminEmail string) *apierror.APIError
}

func (s *stubNotificationClient) SendPaymentActionRequired(ctx context.Context, accountID, adminEmail string) *apierror.APIError {
	if s.sendPaymentActionRequired != nil {
		return s.sendPaymentActionRequired(ctx, accountID, adminEmail)
	}
	return nil
}

type stubAccountUsageRepo struct {
	getAdminEmailByAccountID func(ctx context.Context, accountID string) (string, *apierror.APIError)
}

func (s *stubAccountUsageRepo) GetAdminEmailByAccountID(ctx context.Context, accountID string) (string, *apierror.APIError) {
	if s.getAdminEmailByAccountID != nil {
		return s.getAdminEmailByAccountID(ctx, accountID)
	}
	return "", nil
}

func newTestConsumer(coreClient WebhookCoreClient) *StripeWebhookConsumer {
	return &StripeWebhookConsumer{
		coreClient: coreClient,
	}
}

func newTestConsumerFull(coreClient WebhookCoreClient, notifClient WebhookNotificationClient, accountRepo WebhookAccountUsageRepo) *StripeWebhookConsumer {
	return &StripeWebhookConsumer{
		coreClient:         coreClient,
		notificationClient: notifClient,
		accountUsageRepo:   accountRepo,
	}
}

func TestHandleServicingActivated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	eventID := "evt_1"

	var capturedServiceStatus *string
	var capturedSubID *string
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, customerID string) (string, string, *apierror.APIError) {
			assert.Equal(t, "cus_123", customerID)
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, key, accountID string, _ *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, subID *string, svcStatus *string, _ *string) *apierror.APIError {
			capturedServiceStatus = svcStatus
			capturedSubID = subID
			assert.Equal(t, eventID, key)
			assert.Equal(t, "acct_1", accountID)
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(v2PricingPlanSubscriptionObject{
		ID:       "pps_1",
		Customer: "cus_123",
	})

	err := consumer.handleServicingActivated(ctx, eventID, rawObject)
	require.NoError(t, err)
	assert.Equal(t, "active", *capturedServiceStatus)
	assert.Equal(t, "pps_1", *capturedSubID)
}

func TestHandleServicingCanceled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedServiceStatus *string
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, planCode string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, svcStatus *string, _ *string) *apierror.APIError {
			capturedServiceStatus = svcStatus
			assert.Equal(t, "pro", planCode)
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(v2PricingPlanSubscriptionObject{Customer: "cus_123"})

	err := consumer.handleServicingCanceled(ctx, "evt_2", rawObject)
	require.NoError(t, err)
	assert.Equal(t, "canceled", *capturedServiceStatus)
}

func TestHandleCollectionPaused(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedCollectionStatus *string
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, colStatus *string) *apierror.APIError {
			capturedCollectionStatus = colStatus
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(v2PricingPlanSubscriptionObject{Customer: "cus_123"})

	err := consumer.handleCollectionPaused(ctx, "evt_3", rawObject)
	require.NoError(t, err)
	assert.Equal(t, "paused", *capturedCollectionStatus)
}

func TestHandleCollectionCurrent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedCollectionStatus *string
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, colStatus *string) *apierror.APIError {
			capturedCollectionStatus = colStatus
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(v2PricingPlanSubscriptionObject{Customer: "cus_123"})

	err := consumer.handleCollectionCurrent(ctx, "evt_4", rawObject)
	require.NoError(t, err)
	assert.Equal(t, "current", *capturedCollectionStatus)
}

func TestHandleCadenceErrored(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedStatus *string
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, status *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, _ *string) *apierror.APIError {
			capturedStatus = status
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(v2CadenceObject{ID: "cad_1", Customer: "cus_123"})

	err := consumer.handleCadenceErrored(ctx, "evt_5", rawObject)
	require.NoError(t, err)
	assert.Equal(t, "past_due", *capturedStatus)
}

func TestHandleCustomerDeleted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var clearedAccountID string
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, customerID string) (string, string, *apierror.APIError) {
			assert.Equal(t, "cus_123", customerID)
			return "acct_1", "pro", nil
		},
		clearAccountStripeCustomer: func(_ context.Context, key, accountID string) *apierror.APIError {
			clearedAccountID = accountID
			assert.Equal(t, "evt_6", key)
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(customerObject{ID: "cus_123"})

	err := consumer.handleCustomerDeleted(ctx, "evt_6", rawObject)
	require.NoError(t, err)
	assert.Equal(t, "acct_1", clearedAccountID)
}

func TestHandleServicingActivated_InvalidJSON(t *testing.T) {
	t.Parallel()
	consumer := newTestConsumer(nil)
	err := consumer.handleServicingActivated(context.Background(), "evt_9", json.RawMessage(`{invalid`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestHandleCustomerDeleted_AccountNotFound_NoOp(t *testing.T) {
	t.Parallel()
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "", "", apierror.NewResourceNotFoundError("account not found")
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(customerObject{ID: "cus_orphan"})

	err := consumer.handleCustomerDeleted(context.Background(), "evt_10", rawObject)
	require.NoError(t, err)
}

func TestHandleCustomerDeleted_CoreClientError(t *testing.T) {
	t.Parallel()
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "", "", apierror.NewInternalError(nil, "core unavailable")
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(customerObject{ID: "cus_123"})

	err := consumer.handleCustomerDeleted(context.Background(), "evt_11", rawObject)
	assert.Error(t, err)
}

func TestHandleServicingPaused(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedServiceStatus *string
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, planCode string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, svcStatus *string, _ *string) *apierror.APIError {
			capturedServiceStatus = svcStatus
			assert.Equal(t, "pro", planCode)
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(v2PricingPlanSubscriptionObject{Customer: "cus_123"})

	err := consumer.handleServicingPaused(ctx, "evt_sp", rawObject)
	require.NoError(t, err)
	assert.Equal(t, "paused", *capturedServiceStatus)
}

func TestHandleCollectionAwaitingCustomerAction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedCollectionStatus *string
	var notifAccountID, notifEmail string
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, colStatus *string) *apierror.APIError {
			capturedCollectionStatus = colStatus
			return nil
		},
	}

	notifClient := &stubNotificationClient{
		sendPaymentActionRequired: func(_ context.Context, accountID, adminEmail string) *apierror.APIError {
			notifAccountID = accountID
			notifEmail = adminEmail
			return nil
		},
	}

	accountRepo := &stubAccountUsageRepo{
		getAdminEmailByAccountID: func(_ context.Context, accountID string) (string, *apierror.APIError) {
			assert.Equal(t, "acct_1", accountID)
			return "admin@test.com", nil
		},
	}

	consumer := newTestConsumerFull(client, notifClient, accountRepo)
	rawObject, _ := json.Marshal(v2PricingPlanSubscriptionObject{Customer: "cus_123"})

	err := consumer.handleCollectionAwaitingCustomerAction(ctx, "evt_ca", rawObject)
	require.NoError(t, err)
	assert.Equal(t, "awaiting_customer_action", *capturedCollectionStatus)
	assert.Equal(t, "acct_1", notifAccountID)
	assert.Equal(t, "admin@test.com", notifEmail)
}

func TestHandleCollectionAwaitingCustomerAction_EmailLookupFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, _ *string) *apierror.APIError {
			return nil
		},
	}

	accountRepo := &stubAccountUsageRepo{
		getAdminEmailByAccountID: func(_ context.Context, _ string) (string, *apierror.APIError) {
			return "", apierror.NewInternalError(nil, "db error")
		},
	}

	consumer := newTestConsumerFull(client, &stubNotificationClient{}, accountRepo)
	rawObject, _ := json.Marshal(v2PricingPlanSubscriptionObject{Customer: "cus_123"})

	// Should succeed even if email lookup fails (notification is best-effort)
	err := consumer.handleCollectionAwaitingCustomerAction(ctx, "evt_ca2", rawObject)
	require.NoError(t, err)
}

func TestHandleCadenceBilled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	billedTo := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	var capturedPeriodEnd *time.Time
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, periodEnd *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, _ *string) *apierror.APIError {
			capturedPeriodEnd = periodEnd
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(v2CadenceObject{ID: "cad_1", Customer: "cus_123", BilledTo: &billedTo})

	err := consumer.handleCadenceBilled(ctx, "evt_cb", rawObject)
	require.NoError(t, err)
	require.NotNil(t, capturedPeriodEnd)
	assert.Equal(t, billedTo, *capturedPeriodEnd)
}

func TestHandleCadenceBilled_NilBilledTo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedPeriodEnd *time.Time
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, periodEnd *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, _ *string) *apierror.APIError {
			capturedPeriodEnd = periodEnd
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(v2CadenceObject{ID: "cad_1", Customer: "cus_123"})

	err := consumer.handleCadenceBilled(ctx, "evt_cb2", rawObject)
	require.NoError(t, err)
	assert.Nil(t, capturedPeriodEnd)
}

func TestHandleCadenceCanceled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var capturedStatus *string
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, status *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, _ *string) *apierror.APIError {
			capturedStatus = status
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(v2CadenceObject{ID: "cad_1", Customer: "cus_123"})

	err := consumer.handleCadenceCanceled(ctx, "evt_cc", rawObject)
	require.NoError(t, err)
	assert.Equal(t, "canceled", *capturedStatus)
}

func TestHandleCheckoutSessionCompleted_RecordsPayment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var gotOrderID, gotPI, gotKey string
	called := false
	client := &stubCoreClient{
		recordOrderPayment: func(_ context.Context, key, salesOrderID, paymentIntentID string) *apierror.APIError {
			called = true
			gotKey, gotOrderID, gotPI = key, salesOrderID, paymentIntentID
			return nil
		},
	}

	consumer := newTestConsumer(client)
	rawObject, _ := json.Marshal(checkoutSessionObject{
		ID:            "cs_1",
		PaymentIntent: "pi_1",
		PaymentStatus: "paid",
		Metadata:      map[string]string{"orderID": "or_1"},
	})

	err := consumer.handleCheckoutSessionCompleted(ctx, "evt_co", rawObject)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "evt_co", gotKey)
	assert.Equal(t, "or_1", gotOrderID)
	assert.Equal(t, "pi_1", gotPI)
}

func TestHandleCheckoutSessionCompleted_SkipsWhenNotPaidOrNoOrder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cases := []struct {
		name string
		obj  checkoutSessionObject
	}{
		{"unpaid", checkoutSessionObject{ID: "cs_2", PaymentIntent: "pi_2", PaymentStatus: "unpaid", Metadata: map[string]string{"orderID": "or_2"}}},
		{"no order metadata", checkoutSessionObject{ID: "cs_3", PaymentIntent: "pi_3", PaymentStatus: "paid"}},
		{"no payment intent", checkoutSessionObject{ID: "cs_4", PaymentStatus: "paid", Metadata: map[string]string{"orderID": "or_4"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &stubCoreClient{
				recordOrderPayment: func(_ context.Context, _, _, _ string) *apierror.APIError {
					t.Fatal("RecordOrderPayment should not be called")
					return nil
				},
			}
			consumer := newTestConsumer(client)
			rawObject, _ := json.Marshal(tc.obj)

			err := consumer.handleCheckoutSessionCompleted(ctx, "evt_skip", rawObject)
			require.NoError(t, err)
		})
	}
}
