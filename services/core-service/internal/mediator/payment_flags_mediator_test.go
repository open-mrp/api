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
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
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
	s.settlement.EXPECT().GetInvoicePaymentTotals(ctx, flagsAccountID, []string{"iv_over", "iv_owing", "iv_paid"}).Return([]domain.InvoicePaymentTotals{
		{PaymentTotals: domain.PaymentTotals{ID: "iv_paid", Total: "57.25", Allocated: "57.25"}},
		{PaymentTotals: domain.PaymentTotals{ID: "iv_owing", Total: "57.25", Allocated: "10"}},
		{PaymentTotals: domain.PaymentTotals{ID: "iv_over", Total: "57.25", Allocated: "60"}},
	}, nil)
	s.settlement.EXPECT().UpdateInvoicePaymentStatus(ctx, flagsAccountID, "iv_paid", true, false, false).Return(nil)
	s.settlement.EXPECT().UpdateInvoicePaymentStatus(ctx, flagsAccountID, "iv_owing", false, false, false).Return(nil)
	s.settlement.EXPECT().UpdateInvoicePaymentStatus(ctx, flagsAccountID, "iv_over", true, true, false).Return(nil)

	s.Require().Nil(s.med.Recompute(ctx, flagsAccountID, []string{"tx_spent", "tx_open", "tx_spent"}, []string{"iv_paid", "iv_owing", "iv_over"}))
	s.Empty(s.outbox.messages, "no flag was set by hand, so no one is told")
}

// expectInvoiceRecompute recomputes one invoice and no transactions.
func (s *PaymentFlagsMedTestSuite) expectInvoiceRecompute(inv domain.InvoicePaymentTotals) {
	s.settlement.EXPECT().LockPaymentFlagRows(gomock.Any(), flagsAccountID, []string(nil), []string{inv.ID}).Return(nil)
	s.settlement.EXPECT().GetTransactionAllocationTotals(gomock.Any(), flagsAccountID, []string(nil)).Return(nil, nil)
	s.settlement.EXPECT().UpdateTransactionsFullyAllocated(gomock.Any(), flagsAccountID, []string(nil), gomock.Any()).Return(nil).Times(2)
	s.settlement.EXPECT().GetInvoicePaymentTotals(gomock.Any(), flagsAccountID, []string{inv.ID}).Return([]domain.InvoicePaymentTotals{inv}, nil)
}

func (s *PaymentFlagsMedTestSuite) alerts() []messaging.AlertFanoutData {
	var out []messaging.AlertFanoutData
	for _, msg := range s.outbox.messages {
		if msg.MessageType != string(contracts.NotificationCmdFanout) {
			continue
		}
		var data messaging.AlertFanoutData
		s.Require().NoError(json.Unmarshal(msg.Payload.Data, &data))
		out = append(out, data)
	}
	return out
}

// Someone marked an invoice paid by hand; its payments, recalculated, still leave money owed. The
// recalculation wins, forgets the mark, and tells that person.
func (s *PaymentFlagsMedTestSuite) TestRecomputeOverturnsAHandSetPaidMarkAndTellsWhoSetIt() {
	marker := "us_marker"
	s.expectInvoiceRecompute(domain.InvoicePaymentTotals{
		PaymentTotals: domain.PaymentTotals{ID: "iv_1", Total: "57.25", Allocated: "5"},
		Number:        "INV-1", IsPaidInFull: true, MarkedByID: &marker,
	})
	s.settlement.EXPECT().UpdateInvoicePaymentStatus(gomock.Any(), flagsAccountID, "iv_1", false, false, true).Return(nil)

	s.Require().Nil(s.med.Recompute(context.Background(), flagsAccountID, nil, []string{"iv_1"}))

	alerts := s.alerts()
	s.Require().Len(alerts, 1)
	a := alerts[0]
	s.Equal(flagsAccountID, a.AccountID)
	s.Equal(string(constants.NotificationCategoryInvoicePaymentStatusChanged), a.Category)
	s.Equal([]string{marker}, a.RecipientUserIDs)
	s.Equal(string(constants.ObjectTypeInvoice), a.LinkResourceType)
	s.Equal("iv_1", a.LinkResourceID)
	s.Equal("Invoice INV-1 is no longer marked paid", a.Title)
	s.Contains(a.Body, "$52.25 still owed")
}

func (s *PaymentFlagsMedTestSuite) TestRecomputeOverturnsAHandSetUnpaidMarkAndTellsWhoSetIt() {
	marker := "us_marker"
	s.expectInvoiceRecompute(domain.InvoicePaymentTotals{
		PaymentTotals: domain.PaymentTotals{ID: "iv_1", Total: "57.25", Allocated: "57.25"},
		Number:        "INV-1", IsPaidInFull: false, MarkedByID: &marker,
	})
	s.settlement.EXPECT().UpdateInvoicePaymentStatus(gomock.Any(), flagsAccountID, "iv_1", true, false, true).Return(nil)

	s.Require().Nil(s.med.Recompute(context.Background(), flagsAccountID, nil, []string{"iv_1"}))

	alerts := s.alerts()
	s.Require().Len(alerts, 1)
	s.Equal("Invoice INV-1 is now marked paid", alerts[0].Title)
}

// A mark the recalculation agrees with stands: nothing to tell, and the mark stays the person's.
func (s *PaymentFlagsMedTestSuite) TestRecomputeKeepsAHandSetMarkItAgreesWith() {
	marker := "us_marker"
	s.expectInvoiceRecompute(domain.InvoicePaymentTotals{
		PaymentTotals: domain.PaymentTotals{ID: "iv_1", Total: "57.25", Allocated: "57.25"},
		Number:        "INV-1", IsPaidInFull: true, MarkedByID: &marker,
	})
	s.settlement.EXPECT().UpdateInvoicePaymentStatus(gomock.Any(), flagsAccountID, "iv_1", true, false, false).Return(nil)

	s.Require().Nil(s.med.Recompute(context.Background(), flagsAccountID, nil, []string{"iv_1"}))
	s.Empty(s.alerts())
}

// A flag the last recalculation set is simply recomputed, whichever way it goes.
func (s *PaymentFlagsMedTestSuite) TestRecomputeChangesAFlagNoOneSetByHandSilently() {
	s.expectInvoiceRecompute(domain.InvoicePaymentTotals{
		PaymentTotals: domain.PaymentTotals{ID: "iv_1", Total: "57.25", Allocated: "5"},
		Number:        "INV-1", IsPaidInFull: true,
	})
	s.settlement.EXPECT().UpdateInvoicePaymentStatus(gomock.Any(), flagsAccountID, "iv_1", false, false, false).Return(nil)

	s.Require().Nil(s.med.Recompute(context.Background(), flagsAccountID, nil, []string{"iv_1"}))
	s.Empty(s.alerts())
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
