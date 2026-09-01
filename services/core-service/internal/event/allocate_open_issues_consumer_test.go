package event

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

const (
	pageAccountID = "acct_test"
	pageItemID    = "it_a"
	pageParentMsg = "mg_parent"
)

type continuation struct {
	evt       domain.AllocateOpenIssuesEvent
	messageID string
}

type continuationOutboxRepo struct {
	enqueued *[]continuation
}

func (r continuationOutboxRepo) Create(_ context.Context, input messaging.OutboxMessageInput) (int64, error) {
	if input.MessageType == string(contracts.CoreCmdAllocateOpenIssues) {
		var evt domain.AllocateOpenIssuesEvent
		if err := json.Unmarshal(input.Payload.Data, &evt); err != nil {
			return 0, err
		}
		*r.enqueued = append(*r.enqueued, continuation{evt: evt, messageID: input.MessageID})
	}
	return 0, nil
}

type AllocateOpenIssuesConsumerTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	repoFactory     *factorymock.MockRepoFactory
	reservationRepo *repositorymock.MockInventoryReservationRepo
	consumer        *AllocateOpenIssuesConsumer
	continuations   []continuation
}

func (s *AllocateOpenIssuesConsumerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.continuations = nil

	s.reservationRepo = repositorymock.NewMockInventoryReservationRepo(s.ctrl)
	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)
	s.repoFactory.EXPECT().NewInventoryReservationRepo().Return(s.reservationRepo).AnyTimes()
	// Every per-issue transaction takes the item's ordering root as its first statement.
	s.reservationRepo.EXPECT().LockItemForLedger(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	s.repoFactory.EXPECT().NewOutboxRepo().
		Return(continuationOutboxRepo{enqueued: &s.continuations}).AnyTimes()

	s.consumer = &AllocateOpenIssuesConsumer{
		repos:     s.repoFactory,
		txManager: &stubTxManager{factory: s.repoFactory},
		tracer:    tracing.GetTracer("test.allocate_open_issues_consumer"),
	}
}

func (s *AllocateOpenIssuesConsumerTestSuite) TearDownTest() { s.ctrl.Finish() }

func TestAllocateOpenIssuesConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AllocateOpenIssuesConsumerTestSuite))
}

func (s *AllocateOpenIssuesConsumerTestSuite) refs(n int, at time.Time) []domain.OpenIssueRef {
	out := make([]domain.OpenIssueRef, 0, n)
	for i := range n {
		out = append(out, domain.OpenIssueRef{
			ID:        "ii_" + string(rune('a'+i)),
			CreatedAt: at.Add(time.Duration(i) * time.Minute),
		})
	}
	return out
}

func (s *AllocateOpenIssuesConsumerTestSuite) TestFullPageEnqueuesContinuationFromCursor() {
	at := time.Date(2026, 3, 16, 22, 18, 40, 0, time.UTC)
	refs := s.refs(3, at)

	s.reservationRepo.EXPECT().CountAvailableReceiptsForItem(gomock.Any(), pageAccountID, pageItemID).
		Return(int64(4), nil)
	s.reservationRepo.EXPECT().
		ListOpenIssueIDsForItem(gomock.Any(), pageAccountID, pageItemID, time.Time{}, "", int32(3)).
		Return(refs, nil)
	for _, ref := range refs {
		s.reservationRepo.EXPECT().
			AllocateOneOpenIssue(gomock.Any(), gomock.Any(), pageAccountID, pageItemID, ref.ID).Return(nil)
	}

	err := s.consumer.allocateItem(context.Background(), pageParentMsg, pageAccountID, pageItemID, time.Time{}, "", 3)

	s.Require().Nil(err)
	s.Require().Len(s.continuations, 1)
	s.Equal(refs[2].ID, s.continuations[0].evt.AfterID)
	s.True(s.continuations[0].evt.AfterCreatedAt.Equal(refs[2].CreatedAt))
}

func (s *AllocateOpenIssuesConsumerTestSuite) TestShortPageStopsPaging() {
	at := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	refs := s.refs(2, at)

	s.reservationRepo.EXPECT().CountAvailableReceiptsForItem(gomock.Any(), pageAccountID, pageItemID).
		Return(int64(1), nil)
	s.reservationRepo.EXPECT().
		ListOpenIssueIDsForItem(gomock.Any(), pageAccountID, pageItemID, at, "ii_5", int32(3)).
		Return(refs, nil)
	for _, ref := range refs {
		s.reservationRepo.EXPECT().
			AllocateOneOpenIssue(gomock.Any(), gomock.Any(), pageAccountID, pageItemID, ref.ID).Return(nil)
	}

	err := s.consumer.allocateItem(context.Background(), pageParentMsg, pageAccountID, pageItemID, at, "ii_5", 3)

	s.Require().Nil(err)
	s.Empty(s.continuations)
}

// An item with nothing to draw on must not open a transaction per issue to discover that. The busiest
// items in this database have hundreds of open issues against zero available receipts, and before
// this each of them was a locking scan that found nothing to do.
func (s *AllocateOpenIssuesConsumerTestSuite) TestNoAvailableReceiptsDoesNoWork() {
	s.reservationRepo.EXPECT().CountAvailableReceiptsForItem(gomock.Any(), pageAccountID, pageItemID).
		Return(int64(0), nil)

	err := s.consumer.allocateItem(context.Background(), pageParentMsg, pageAccountID, pageItemID, time.Time{}, "", 3)

	s.Require().Nil(err)
	s.Empty(s.continuations)
}

// One issue failing is not the page failing. Its own transaction rolls back and the row stays open,
// but the issues behind it still get their turn — stopping at the first failure lets one poisoned
// issue starve every issue after it, which is how a whole item's demand goes uncovered indefinitely.
//
// The failure is still returned, so message_inbox records it. Swallowing it would remove the only
// signal that surfaced this incident.
func (s *AllocateOpenIssuesConsumerTestSuite) TestOneFailingIssueDoesNotStarveTheRest() {
	at := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	refs := s.refs(3, at)

	s.reservationRepo.EXPECT().CountAvailableReceiptsForItem(gomock.Any(), pageAccountID, pageItemID).
		Return(int64(2), nil)
	s.reservationRepo.EXPECT().
		ListOpenIssueIDsForItem(gomock.Any(), pageAccountID, pageItemID, time.Time{}, "", int32(3)).
		Return(refs, nil)

	s.reservationRepo.EXPECT().AllocateOneOpenIssue(gomock.Any(), gomock.Any(), pageAccountID, pageItemID, refs[0].ID).Return(nil)
	s.reservationRepo.EXPECT().AllocateOneOpenIssue(gomock.Any(), gomock.Any(), pageAccountID, pageItemID, refs[1].ID).
		Return(apierror.NewInternalError(nil, "boom"))
	s.reservationRepo.EXPECT().AllocateOneOpenIssue(gomock.Any(), gomock.Any(), pageAccountID, pageItemID, refs[2].ID).Return(nil)

	err := s.consumer.allocateItem(context.Background(), pageParentMsg, pageAccountID, pageItemID, time.Time{}, "", 3)

	s.Require().NotNil(err, "the page must still report the failure so message_inbox records it")
	// The page was full, so the chain continues past the failed issue rather than stalling on it.
	s.Require().Len(s.continuations, 1)
	s.Equal(refs[2].ID, s.continuations[0].evt.AfterID)
}

// The continuation carries an id derived from the delivery that produced it.
//
// A retry of that delivery republishes the same id, and the inbox's (handler, message_id) key makes
// it a no-op — without which four delivery attempts become four chains for one item, each free to
// fork again. Keyed on the cursor alone the id would instead be stable for all time, so the second
// chain ever to reach a page would be deduped against the first and stop there, leaving everything
// past it uncovered. Both properties are asserted here because losing either is silent.
func (s *AllocateOpenIssuesConsumerTestSuite) TestContinuationIDIsStablePerDeliveryAndNotAcrossChains() {
	at := time.Date(2026, 3, 16, 22, 18, 40, 0, time.UTC)

	same := continuationMessageID(pageParentMsg, pageAccountID, pageItemID, at, "ii_last")
	s.Equal(same, continuationMessageID(pageParentMsg, pageAccountID, pageItemID, at, "ii_last"),
		"a retry of the same delivery must republish the same continuation, not fork the chain")

	s.NotEqual(same, continuationMessageID("mg_other_chain", pageAccountID, pageItemID, at, "ii_last"),
		"a new chain reaching the same page must get its own id, or it is deduped against the old one and stops")
	s.NotEqual(same, continuationMessageID(pageParentMsg, pageAccountID, pageItemID, at, "ii_other"),
		"a different cursor is a different continuation")
	s.NotEqual(same, continuationMessageID(pageParentMsg, pageAccountID, "it_b", at, "ii_last"),
		"a different item is a different continuation")
}

func (s *AllocateOpenIssuesConsumerTestSuite) TestContinuationIDIsCarriedOntoTheOutboxMessage() {
	at := time.Date(2026, 3, 16, 22, 18, 40, 0, time.UTC)
	refs := s.refs(1, at)

	s.reservationRepo.EXPECT().CountAvailableReceiptsForItem(gomock.Any(), pageAccountID, pageItemID).
		Return(int64(1), nil)
	s.reservationRepo.EXPECT().
		ListOpenIssueIDsForItem(gomock.Any(), pageAccountID, pageItemID, time.Time{}, "", int32(1)).
		Return(refs, nil)
	s.reservationRepo.EXPECT().
		AllocateOneOpenIssue(gomock.Any(), gomock.Any(), pageAccountID, pageItemID, refs[0].ID).Return(nil)

	err := s.consumer.allocateItem(context.Background(), pageParentMsg, pageAccountID, pageItemID, time.Time{}, "", 1)

	s.Require().Nil(err)
	s.Require().Len(s.continuations, 1)
	s.Equal(
		continuationMessageID(pageParentMsg, pageAccountID, pageItemID, refs[0].CreatedAt, refs[0].ID),
		s.continuations[0].messageID,
		"the continuation must go out under its derived id, or the outbox mints a random one and retries fork")
}

// The producers that START a chain must not carry a derived id. Keyed on (item, epoch cursor) it
// would be identical for every enqueue that item ever gets, so the inbox would dedupe all of them
// against the first and the item would never be allocated again.
func (s *AllocateOpenIssuesConsumerTestSuite) TestChainStartersEnqueueWithoutADerivedID() {
	err := mediator.EnqueueAllocateOpenIssuesFrom(context.Background(),
		continuationOutboxRepo{enqueued: &s.continuations}, pageAccountID, pageItemID, time.Time{}, "", "")

	s.Require().Nil(err)
	s.Require().Len(s.continuations, 1)
	s.Empty(s.continuations[0].messageID,
		"a chain starter must leave the id empty so the outbox mints a fresh one per enqueue")
}
