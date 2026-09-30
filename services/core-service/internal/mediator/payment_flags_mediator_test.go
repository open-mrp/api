package mediator

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
)

const flagsAccountID = "ac_flags"

type PaymentFlagsMedTestSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	settlement *repositorymock.MockSettlementRepo
	outbox     *recordingOutboxRepo
	med        domain.PaymentFlagsMed
}

func (s *PaymentFlagsMedTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.settlement = repositorymock.NewMockSettlementRepo(s.ctrl)
	s.outbox = &recordingOutboxRepo{}
	factory := factorymock.NewMockRepoFactory(s.ctrl)
	factory.EXPECT().NewSettlementRepo().Return(s.settlement).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(s.outbox).AnyTimes()
	s.med = NewPaymentFlagsMed(&PaymentFlagsMedConfig{Repos: factory})
}

func (s *PaymentFlagsMedTestSuite) TearDownTest() { s.ctrl.Finish() }

func TestPaymentFlagsMedTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PaymentFlagsMedTestSuite))
}

func (s *PaymentFlagsMedTestSuite) TestEnqueueWritesOneDeduplicatedCommand() {
	s.Require().Nil(s.med.Enqueue(context.Background(), flagsAccountID, []string{"tx_b", "tx_a", "tx_b"}, []string{"iv_1", "iv_1", ""}))

	s.Require().Len(s.outbox.messages, 1)
	msg := s.outbox.messages[0]
	s.Equal(string(contracts.CoreCmdRecomputePaymentFlags), msg.MessageType)
	s.Equal(string(contracts.CoreCmdRecomputePaymentFlags), msg.RoutingKey)
	var evt domain.RecomputePaymentFlagsEvent
	s.Require().NoError(json.Unmarshal(msg.Payload.Data, &evt))
	s.Equal(domain.RecomputePaymentFlagsEvent{AccountID: flagsAccountID, TransactionIDs: []string{"tx_a", "tx_b"}, InvoiceIDs: []string{"iv_1"}}, evt)
}

func (s *PaymentFlagsMedTestSuite) TestEnqueueWithNothingToRecomputeWritesNothing() {
	s.Require().Nil(s.med.Enqueue(context.Background(), flagsAccountID, nil, []string{""}))
	s.Empty(s.outbox.messages)
}

func (s *PaymentFlagsMedTestSuite) TestRecomputeLocksBeforeReadingAndAppliesTheDashboardRules() {
	ctx := context.Background()
	gomock.InOrder(
		s.settlement.EXPECT().LockPaymentFlagRows(ctx, flagsAccountID, []string{"tx_open", "tx_spent"}, []string{"iv_over", "iv_owing", "iv_paid"}).Return(nil),
		s.settlement.EXPECT().GetTransactionAllocationTotals(ctx, flagsAccountID, []string{"tx_open", "tx_spent"}).Return([]domain.PaymentTotals{
			{ID: "tx_spent", Total: "100", Allocated: "99.996"}, // rounds to nothing left
			{ID: "tx_open", Total: "100", Allocated: "99.995"},  // half a cent rounds up to a cent left
		}, nil),
	)
	s.settlement.EXPECT().UpdateTransactionsFullyAllocated(ctx, flagsAccountID, []string{"tx_spent"}, true).Return(nil)
	s.settlement.EXPECT().UpdateTransactionsFullyAllocated(ctx, flagsAccountID, []string{"tx_open"}, false).Return(nil)
	s.settlement.EXPECT().GetInvoicePaymentTotals(ctx, flagsAccountID, []string{"iv_over", "iv_owing", "iv_paid"}).Return([]domain.PaymentTotals{
		{ID: "iv_paid", Total: "57.25", Allocated: "57.25"},
		{ID: "iv_owing", Total: "57.25", Allocated: "10"},
		{ID: "iv_over", Total: "57.25", Allocated: "60"},
	}, nil)
	s.settlement.EXPECT().UpdateInvoicePaymentStatus(ctx, flagsAccountID, "iv_paid", true, false).Return(nil)
	s.settlement.EXPECT().UpdateInvoicePaymentStatus(ctx, flagsAccountID, "iv_owing", false, false).Return(nil)
	s.settlement.EXPECT().UpdateInvoicePaymentStatus(ctx, flagsAccountID, "iv_over", true, true).Return(nil)

	s.Require().Nil(s.med.Recompute(ctx, flagsAccountID, []string{"tx_spent", "tx_open", "tx_spent"}, []string{"iv_paid", "iv_owing", "iv_over"}))
}

func (s *PaymentFlagsMedTestSuite) TestRecomputeStopsWhenTheLockFails() {
	s.settlement.EXPECT().LockPaymentFlagRows(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(apierror.NewInternalError(nil, "lock wait timeout"))

	s.Require().NotNil(s.med.Recompute(context.Background(), flagsAccountID, []string{"tx_1"}, nil))
}

func (s *PaymentFlagsMedTestSuite) TestRecomputeIgnoresRowsThatNoLongerExist() {
	// A deleted adjustment is still named by the command; it simply has no totals to recompute.
	s.settlement.EXPECT().LockPaymentFlagRows(gomock.Any(), flagsAccountID, []string{"tx_gone"}, []string(nil)).Return(nil)
	s.settlement.EXPECT().GetTransactionAllocationTotals(gomock.Any(), flagsAccountID, []string{"tx_gone"}).Return(nil, nil)
	s.settlement.EXPECT().UpdateTransactionsFullyAllocated(gomock.Any(), flagsAccountID, []string(nil), gomock.Any()).Return(nil).Times(2)
	s.settlement.EXPECT().GetInvoicePaymentTotals(gomock.Any(), flagsAccountID, []string(nil)).Return(nil, nil)

	s.Require().Nil(s.med.Recompute(context.Background(), flagsAccountID, []string{"tx_gone"}, nil))
}
