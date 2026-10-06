package event

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

type fakeBatchUsageRepo struct{ customerID *string }

func (f fakeBatchUsageRepo) GetStripeCustomerIDByAccountID(context.Context, string) (*string, *apierror.APIError) {
	return f.customerID, nil
}

type fakeBatchMeterClient struct {
	err                              error
	calls                            int
	eventName, customer, idempotency string
}

func (f *fakeBatchMeterClient) ReportMeterEvent(_ context.Context, eventName, customer string, _ int, idempotencyKey string) error {
	f.calls++
	f.eventName, f.customer, f.idempotency = eventName, customer, idempotencyKey
	return f.err
}

type permanentErr struct{}

func (permanentErr) Error() string   { return "no active meter" }
func (permanentErr) Permanent() bool { return true }

func batchCreatedDelivery(t *testing.T) amqp.Delivery {
	t.Helper()
	data, err := json.Marshal(messaging.BatchCreatedReportData{AccountID: "ac_1", BatchID: "bt_1"})
	require.NoError(t, err)
	body, err := json.Marshal(contracts.AmqpMessage{Data: data})
	require.NoError(t, err)
	return amqp.Delivery{Body: body, MessageId: "mg_1"}
}

func TestBatchCreatedHandler(t *testing.T) {
	t.Parallel()
	customer := "cus_1"

	t.Run("meters one batch against the account's customer, keyed by the message", func(t *testing.T) {
		client := &fakeBatchMeterClient{}
		require.NoError(t, NewBatchCreatedHandler(fakeBatchUsageRepo{&customer}, client).Handle(context.Background(), batchCreatedDelivery(t)))
		assert.Equal(t, 1, client.calls)
		assert.Equal(t, "openmrp_batches", client.eventName)
		assert.Equal(t, customer, client.customer)
		assert.Equal(t, "mg_1", client.idempotency)
	})

	t.Run("skips an account with no Stripe customer", func(t *testing.T) {
		client := &fakeBatchMeterClient{}
		require.NoError(t, NewBatchCreatedHandler(fakeBatchUsageRepo{}, client).Handle(context.Background(), batchCreatedDelivery(t)))
		assert.Zero(t, client.calls)
	})

	t.Run("a refusal Stripe will repeat is not retried", func(t *testing.T) {
		client := &fakeBatchMeterClient{err: permanentErr{}}
		err := NewBatchCreatedHandler(fakeBatchUsageRepo{&customer}, client).Handle(context.Background(), batchCreatedDelivery(t))
		assert.ErrorIs(t, err, errBatchMeterRejected)
	})

	t.Run("a transient failure is retried", func(t *testing.T) {
		client := &fakeBatchMeterClient{err: errors.New("connection reset")}
		err := NewBatchCreatedHandler(fakeBatchUsageRepo{&customer}, client).Handle(context.Background(), batchCreatedDelivery(t))
		require.Error(t, err)
		assert.NotErrorIs(t, err, errBatchMeterRejected)
	})
}
