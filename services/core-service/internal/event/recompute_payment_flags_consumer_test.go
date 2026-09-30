package event

import (
	"context"
	"encoding/json"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

func paymentFlagsDelivery(t *testing.T, envelope contracts.AmqpMessage, evt domain.RecomputePaymentFlagsEvent) amqp.Delivery {
	t.Helper()
	data, err := json.Marshal(evt)
	require.NoError(t, err)
	envelope.Data = data
	body, err := json.Marshal(envelope)
	require.NoError(t, err)
	return amqp.Delivery{Body: body, RoutingKey: string(contracts.CoreCmdRecomputePaymentFlags), MessageId: "msg_flags"}
}

func newPaymentFlagsConsumer(t *testing.T) (*RecomputePaymentFlagsConsumer, *repositorymock.MockSettlementRepo, *stubTxManager) {
	ctrl := gomock.NewController(t)
	settlement := repositorymock.NewMockSettlementRepo(ctrl)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewSettlementRepo().Return(settlement).AnyTimes()
	tx := &stubTxManager{factory: factory}
	return &RecomputePaymentFlagsConsumer{
		inboxConsumer: messaging.NewInboxConsumer(nil, "core-service"),
		txManager:     tx,
		tracer:        tracing.GetTracer("test.recompute_payment_flags_consumer"),
	}, settlement, tx
}

func TestRecomputePaymentFlagsConsumerRecomputesInOneTransaction(t *testing.T) {
	t.Parallel()
	consumer, settlement, tx := newPaymentFlagsConsumer(t)

	settlement.EXPECT().LockPaymentFlagRows(gomock.Any(), "ac_1", []string{"tx_1"}, []string{"iv_1"}).Return(nil)
	settlement.EXPECT().GetTransactionAllocationTotals(gomock.Any(), "ac_1", []string{"tx_1"}).
		Return([]domain.PaymentTotals{{ID: "tx_1", Total: "10", Allocated: "10"}}, nil)
	settlement.EXPECT().UpdateTransactionsFullyAllocated(gomock.Any(), "ac_1", []string{"tx_1"}, true).Return(nil)
	settlement.EXPECT().UpdateTransactionsFullyAllocated(gomock.Any(), "ac_1", []string(nil), false).Return(nil)
	settlement.EXPECT().GetInvoicePaymentTotals(gomock.Any(), "ac_1", []string{"iv_1"}).
		Return([]domain.PaymentTotals{{ID: "iv_1", Total: "10", Allocated: "10"}}, nil)
	settlement.EXPECT().UpdateInvoicePaymentStatus(gomock.Any(), "ac_1", "iv_1", true, false).Return(nil)

	err := consumer.handleMessage(context.Background(), paymentFlagsDelivery(t, contracts.AmqpMessage{},
		domain.RecomputePaymentFlagsEvent{AccountID: "ac_1", TransactionIDs: []string{"tx_1"}, InvoiceIDs: []string{"iv_1"}}))

	require.NoError(t, err)
	require.Equal(t, 1, tx.calls, "the recompute and the inbox completion share one transaction")
}

func TestRecomputePaymentFlagsConsumerDiscardsCommandsItCannotRun(t *testing.T) {
	t.Parallel()
	cases := map[string]domain.RecomputePaymentFlagsEvent{
		"no account":        {TransactionIDs: []string{"tx_1"}},
		"nothing to update": {AccountID: "ac_1"},
	}
	for name, evt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			consumer, _, tx := newPaymentFlagsConsumer(t)
			require.NoError(t, consumer.handleMessage(context.Background(), paymentFlagsDelivery(t, contracts.AmqpMessage{}, evt)))
			require.Zero(t, tx.calls, "a malformed command opens no transaction")
		})
	}
}

func TestRecomputePaymentFlagsConsumerRejectsAnUnreadableBody(t *testing.T) {
	t.Parallel()
	consumer, _, _ := newPaymentFlagsConsumer(t)
	require.Error(t, consumer.handleMessage(context.Background(), amqp.Delivery{Body: []byte("not json")}))
}
