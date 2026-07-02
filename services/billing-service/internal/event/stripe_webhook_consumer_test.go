package event

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubEventLogRepo implements EventLogRepo for tests.
type stubEventLogRepo struct {
	existsFn func(ctx context.Context, eventID, objectID string) (bool, *apierror.APIError)
	insertFn func(ctx context.Context, eventID, eventType, objectID string) *apierror.APIError
}

func (s *stubEventLogRepo) Exists(ctx context.Context, eventID, objectID string) (bool, *apierror.APIError) {
	if s.existsFn != nil {
		return s.existsFn(ctx, eventID, objectID)
	}
	return false, nil
}

func (s *stubEventLogRepo) Insert(ctx context.Context, eventID, eventType, objectID string) *apierror.APIError {
	if s.insertFn != nil {
		return s.insertFn(ctx, eventID, eventType, objectID)
	}
	return nil
}

// stubStripeClient implements domain.StripeClient for consumer tests.
type stubStripeClient struct {
	fetchObjectFn func(ctx context.Context, objectURL string) ([]byte, error)
}

func (s *stubStripeClient) VerifyWebhookSignature([]byte, string) (*domain.StripeEvent, error) {
	return nil, nil
}
func (s *stubStripeClient) CreateCustomer(context.Context, string, string, string, map[string]string) (*domain.StripeCustomer, error) {
	return nil, nil
}
func (s *stubStripeClient) CreateBillingPortalSession(context.Context, string, string) (*domain.StripeBillingPortalSession, error) {
	return nil, nil
}
func (s *stubStripeClient) GetPricingPlan(context.Context, string) (*domain.StripePricingPlan, error) {
	return nil, nil
}
func (s *stubStripeClient) CreateBillingProfile(context.Context, string, string) (string, error) {
	return "", nil
}
func (s *stubStripeClient) CreateBillingCadence(context.Context, string, string) (string, error) {
	return "", nil
}
func (s *stubStripeClient) CreateBillingIntent(context.Context, string, []domain.BillingIntentAction, string) (string, error) {
	return "", nil
}
func (s *stubStripeClient) ReserveBillingIntent(context.Context, string) (*domain.BillingIntentReservation, error) {
	return nil, nil
}
func (s *stubStripeClient) CreatePaymentIntent(context.Context, int64, string, string, string) (string, error) {
	return "", nil
}
func (s *stubStripeClient) CommitBillingIntent(context.Context, string, *string, string) (*domain.BillingIntentCommitResult, error) {
	return nil, nil
}
func (s *stubStripeClient) VoidBillingIntent(context.Context, string) error { return nil }
func (s *stubStripeClient) CreateSetupIntent(context.Context, string, string) (*domain.StripeSetupIntent, error) {
	return nil, nil
}
func (s *stubStripeClient) GetSetupIntent(context.Context, string) (*domain.StripeSetupIntent, error) {
	return nil, nil
}
func (s *stubStripeClient) ReportMeterEvent(context.Context, string, string, int, string) error {
	return nil
}

func (s *stubStripeClient) GetAgentTokenSpendCents(context.Context, string, string, time.Time) (int64, error) {
	return 0, nil
}

func (s *stubStripeClient) FetchObject(ctx context.Context, objectURL string) ([]byte, error) {
	if s.fetchObjectFn != nil {
		return s.fetchObjectFn(ctx, objectURL)
	}
	return nil, nil
}

func buildDelivery(eventID, eventType, objectID string) amqp.Delivery {
	eventData := map[string]any{
		"id":   eventID,
		"type": eventType,
		"data": map[string]any{
			"object": map[string]any{
				"id":       objectID,
				"customer": "cus_test",
			},
		},
	}
	eventBytes, _ := json.Marshal(eventData)
	amqpMsg := contracts.AmqpMessage{Data: eventBytes}
	body, _ := json.Marshal(amqpMsg)
	return amqp.Delivery{Body: body}
}

func buildV2Delivery(eventID, eventType, relatedObjectID, relatedURL string) amqp.Delivery {
	eventData := map[string]any{
		"id":   eventID,
		"type": eventType,
		"related_object": map[string]any{
			"id":   relatedObjectID,
			"type": "pricing_plan_subscription",
			"url":  relatedURL,
		},
	}
	eventBytes, _ := json.Marshal(eventData)
	amqpMsg := contracts.AmqpMessage{Data: eventBytes}
	body, _ := json.Marshal(amqpMsg)
	return amqp.Delivery{Body: body}
}

func newConsumerForDispatch(coreClient WebhookCoreClient, eventLog EventLogRepo, stripeClient domain.StripeClient) *StripeWebhookConsumer {
	return newConsumerForDispatchFull(coreClient, eventLog, stripeClient, &stubNotificationClient{}, &stubAccountUsageRepo{})
}

func newConsumerForDispatchFull(coreClient WebhookCoreClient, eventLog EventLogRepo, stripeClient domain.StripeClient, notifClient WebhookNotificationClient, accountRepo WebhookAccountUsageRepo) *StripeWebhookConsumer {
	return &StripeWebhookConsumer{
		coreClient:         coreClient,
		stripeEventLogRepo: eventLog,
		stripeClient:       stripeClient,
		notificationClient: notifClient,
		accountUsageRepo:   accountRepo,
		tracer:             tracing.GetTracer("test"),
	}
}

func TestHandleStripeWebhook_V1CustomerDeleted(t *testing.T) {
	t.Parallel()
	var handlerCalled bool
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		clearAccountStripeCustomer: func(_ context.Context, _, _ string) *apierror.APIError {
			handlerCalled = true
			return nil
		},
	}

	var insertCalled bool
	eventLog := &stubEventLogRepo{
		insertFn: func(_ context.Context, eventID, eventType, objectID string) *apierror.APIError {
			insertCalled = true
			assert.Equal(t, "evt_del", eventID)
			assert.Equal(t, "customer.deleted", eventType)
			return nil
		},
	}

	consumer := newConsumerForDispatch(client, eventLog, &stubStripeClient{})
	msg := buildDelivery("evt_del", "customer.deleted", "cus_test")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.True(t, insertCalled)
}

func TestHandleStripeWebhook_V2ServicingActivated(t *testing.T) {
	t.Parallel()
	var handlerCalled bool
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, svcStatus *string, _ *string) *apierror.APIError {
			handlerCalled = true
			assert.Equal(t, "active", *svcStatus)
			return nil
		},
	}

	subData, _ := json.Marshal(v2PricingPlanSubscriptionObject{
		ID:       "pps_1",
		Customer: "cus_test",
	})

	stripeClient := &stubStripeClient{
		fetchObjectFn: func(_ context.Context, url string) ([]byte, error) {
			assert.Equal(t, "/v2/billing/subscriptions/pps_1", url)
			return subData, nil
		},
	}

	consumer := newConsumerForDispatch(client, &stubEventLogRepo{}, stripeClient)
	msg := buildV2Delivery("evt_v2", "v2.billing.pricing_plan_subscription.servicing_activated", "pps_1", "/v2/billing/subscriptions/pps_1")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestHandleStripeWebhook_DuplicateEventSkipped(t *testing.T) {
	t.Parallel()
	eventLog := &stubEventLogRepo{
		existsFn: func(_ context.Context, eventID, objectID string) (bool, *apierror.APIError) {
			return true, nil
		},
	}

	consumer := newConsumerForDispatch(nil, eventLog, &stubStripeClient{})
	msg := buildDelivery("evt_dup", "customer.deleted", "cus_123")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	require.NoError(t, err)
	// Handler should not be called because event was deduplicated
}

func TestHandleStripeWebhook_UnknownEventType(t *testing.T) {
	t.Parallel()
	consumer := newConsumerForDispatch(nil, &stubEventLogRepo{}, &stubStripeClient{})
	msg := buildDelivery("evt_unk", "unknown.event.type", "obj_1")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	assert.NoError(t, err)
}

func TestHandleStripeWebhook_InsertCalledAfterSuccess(t *testing.T) {
	t.Parallel()
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		clearAccountStripeCustomer: func(_ context.Context, _, _ string) *apierror.APIError {
			return nil
		},
	}

	var insertArgs struct {
		eventID, eventType, objectID string
	}
	eventLog := &stubEventLogRepo{
		insertFn: func(_ context.Context, eventID, eventType, objectID string) *apierror.APIError {
			insertArgs.eventID = eventID
			insertArgs.eventType = eventType
			insertArgs.objectID = objectID
			return nil
		},
	}

	consumer := newConsumerForDispatch(client, eventLog, &stubStripeClient{})
	msg := buildDelivery("evt_ok", "customer.deleted", "cus_456")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, "evt_ok", insertArgs.eventID)
	assert.Equal(t, "customer.deleted", insertArgs.eventType)
	assert.Equal(t, "cus_456", insertArgs.objectID)
}

func TestHandleStripeWebhook_HandlerError_NoInsert(t *testing.T) {
	t.Parallel()
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "", "", apierror.NewInternalError(fmt.Errorf("fail"), "core error")
		},
	}

	var insertCalled bool
	eventLog := &stubEventLogRepo{
		insertFn: func(_ context.Context, _, _, _ string) *apierror.APIError {
			insertCalled = true
			return nil
		},
	}

	consumer := newConsumerForDispatch(client, eventLog, &stubStripeClient{})
	msg := buildDelivery("evt_fail", "customer.deleted", "cus_789")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	assert.Error(t, err)
	assert.False(t, insertCalled)
}

func TestHandleStripeWebhook_V2FetchObjectCalled(t *testing.T) {
	t.Parallel()
	var fetchedURL string
	stripeClient := &stubStripeClient{
		fetchObjectFn: func(_ context.Context, url string) ([]byte, error) {
			fetchedURL = url
			return json.Marshal(v2PricingPlanSubscriptionObject{
				ID:       "pps_fetch",
				Customer: "cus_test",
			})
		},
	}

	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, _ *string) *apierror.APIError {
			return nil
		},
	}

	consumer := newConsumerForDispatch(client, &stubEventLogRepo{}, stripeClient)
	msg := buildV2Delivery("evt_fetch", "v2.billing.pricing_plan_subscription.collection_current", "pps_fetch", "/v2/billing/subscriptions/pps_fetch")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, "/v2/billing/subscriptions/pps_fetch", fetchedURL)
}

func TestHandleStripeWebhook_V2ServicingPaused(t *testing.T) {
	t.Parallel()
	var handlerCalled bool
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, svcStatus *string, _ *string) *apierror.APIError {
			handlerCalled = true
			assert.Equal(t, "paused", *svcStatus)
			return nil
		},
	}

	subData, _ := json.Marshal(v2PricingPlanSubscriptionObject{ID: "pps_1", Customer: "cus_test"})
	stripeClient := &stubStripeClient{
		fetchObjectFn: func(_ context.Context, _ string) ([]byte, error) { return subData, nil },
	}

	consumer := newConsumerForDispatch(client, &stubEventLogRepo{}, stripeClient)
	msg := buildV2Delivery("evt_sp", "v2.billing.pricing_plan_subscription.servicing_paused", "pps_1", "/v2/billing/subscriptions/pps_1")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestHandleStripeWebhook_V2CollectionAwaitingCustomerAction(t *testing.T) {
	t.Parallel()
	var handlerCalled bool
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, colStatus *string) *apierror.APIError {
			handlerCalled = true
			assert.Equal(t, "awaiting_customer_action", *colStatus)
			return nil
		},
	}

	subData, _ := json.Marshal(v2PricingPlanSubscriptionObject{ID: "pps_1", Customer: "cus_test"})
	stripeClient := &stubStripeClient{
		fetchObjectFn: func(_ context.Context, _ string) ([]byte, error) { return subData, nil },
	}
	accountRepo := &stubAccountUsageRepo{
		getAdminEmailByAccountID: func(_ context.Context, _ string) (string, *apierror.APIError) {
			return "admin@test.com", nil
		},
	}

	consumer := newConsumerForDispatchFull(client, &stubEventLogRepo{}, stripeClient, &stubNotificationClient{}, accountRepo)
	msg := buildV2Delivery("evt_ca", "v2.billing.pricing_plan_subscription.collection_awaiting_customer_action", "pps_1", "/v2/billing/subscriptions/pps_1")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestHandleStripeWebhook_V2CadenceBilled(t *testing.T) {
	t.Parallel()
	var handlerCalled bool
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, _ *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, _ *string) *apierror.APIError {
			handlerCalled = true
			return nil
		},
	}

	cadenceData, _ := json.Marshal(v2CadenceObject{ID: "cad_1", Customer: "cus_test"})
	stripeClient := &stubStripeClient{
		fetchObjectFn: func(_ context.Context, _ string) ([]byte, error) { return cadenceData, nil },
	}

	consumer := newConsumerForDispatch(client, &stubEventLogRepo{}, stripeClient)
	msg := buildV2Delivery("evt_cb", "v2.billing.cadence.billed", "cad_1", "/v2/billing/cadences/cad_1")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestHandleStripeWebhook_V2CadenceCanceled(t *testing.T) {
	t.Parallel()
	var handlerCalled bool
	client := &stubCoreClient{
		getAccountByStripeCustomerID: func(_ context.Context, _ string) (string, string, *apierror.APIError) {
			return "acct_1", "pro", nil
		},
		updateAccountSubscription: func(_ context.Context, _, _ string, status *string, _ string, _ *string, _ *time.Time, _ *string, _ *string, _ *string, _ *string, _ *string, _ *string) *apierror.APIError {
			handlerCalled = true
			assert.Equal(t, "canceled", *status)
			return nil
		},
	}

	cadenceData, _ := json.Marshal(v2CadenceObject{ID: "cad_1", Customer: "cus_test"})
	stripeClient := &stubStripeClient{
		fetchObjectFn: func(_ context.Context, _ string) ([]byte, error) { return cadenceData, nil },
	}

	consumer := newConsumerForDispatch(client, &stubEventLogRepo{}, stripeClient)
	msg := buildV2Delivery("evt_cc", "v2.billing.cadence.canceled", "cad_1", "/v2/billing/cadences/cad_1")

	err := consumer.handleStripeWebhook(context.Background(), msg)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}
