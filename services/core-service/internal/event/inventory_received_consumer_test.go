package event

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	"github.com/open-mrp/api/shared/messaging"
)

const allocAccountID = "acct_test"

// failingOutboxRepo refuses the enqueue for one item, standing in for a write that fails mid-run.
type failingOutboxRepo struct {
	recordingOutboxRepo
	failItemID string
}

func (r failingOutboxRepo) Create(ctx context.Context, input messaging.OutboxMessageInput) (int64, error) {
	failed := false
	inspect := recordingOutboxRepo{onAllocateOpenIssues: func(itemID string) {
		failed = itemID == r.failItemID
	}}
	if _, err := inspect.Create(ctx, input); err != nil {
		return 0, err
	}
	if failed {
		return 0, errors.New("boom")
	}
	return r.recordingOutboxRepo.Create(ctx, input)
}

type InventoryReceivedConsumerTestSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	repoFactory *factorymock.MockRepoFactory
	// allocated records, in order, the items whose open-issue allocation the event enqueued.
	allocated []string
}

func (s *InventoryReceivedConsumerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.allocated = nil

	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)
}

func (s *InventoryReceivedConsumerTestSuite) TearDownTest() { s.ctrl.Finish() }

func TestInventoryReceivedConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InventoryReceivedConsumerTestSuite))
}

func (s *InventoryReceivedConsumerTestSuite) recorder() recordingOutboxRepo {
	return recordingOutboxRepo{
		onAllocateOpenIssues: func(itemID string) { s.allocated = append(s.allocated, itemID) },
	}
}

func (s *InventoryReceivedConsumerTestSuite) expectOutbox() {
	s.repoFactory.EXPECT().NewOutboxRepo().Return(s.recorder()).AnyTimes()
}

// Every item whose stock moved is offered to the demand waiting on it, in the order given.
func (s *InventoryReceivedConsumerTestSuite) TestEnqueuesEveryItem() {
	s.expectOutbox()

	err := enqueueAllocationForItems(context.Background(), s.repoFactory, allocAccountID,
		[]string{"it_a", "it_b", "it_c"})

	s.Require().Nil(err)
	s.Equal([]string{"it_a", "it_b", "it_c"}, s.allocated)
}

// One cause routinely names an item twice — a step consuming a material in two places. Allocating
// twice for one arrival walks the same open issues again for nothing.
func (s *InventoryReceivedConsumerTestSuite) TestEnqueuesEachItemOnce() {
	s.expectOutbox()

	err := enqueueAllocationForItems(context.Background(), s.repoFactory, allocAccountID,
		[]string{"it_a", "it_b", "it_a", "it_a"})

	s.Require().Nil(err)
	s.Equal([]string{"it_a", "it_b"}, s.allocated)
}

func (s *InventoryReceivedConsumerTestSuite) TestIgnoresEmptyItemIDs() {
	s.expectOutbox()

	err := enqueueAllocationForItems(context.Background(), s.repoFactory, allocAccountID,
		[]string{"", "it_a", ""})

	s.Require().Nil(err)
	s.Equal([]string{"it_a"}, s.allocated)
}

func (s *InventoryReceivedConsumerTestSuite) TestNoItemsIsNoWork() {
	s.expectOutbox()

	err := enqueueAllocationForItems(context.Background(), s.repoFactory, allocAccountID, nil)

	s.Require().Nil(err)
	s.Empty(s.allocated)
}

// A failure stops the run rather than continuing, so the whole transaction rolls back and the
// delivery is retried. Enqueuing the rest and reporting success would leave the failed item short
// with nothing recording that allocation was still owed — which is the failure mode the nightly
// sweep used to paper over.
func (s *InventoryReceivedConsumerTestSuite) TestStopsAndReportsOnFailure() {
	s.repoFactory.EXPECT().NewOutboxRepo().
		Return(failingOutboxRepo{recordingOutboxRepo: s.recorder(), failItemID: "it_b"}).AnyTimes()

	err := enqueueAllocationForItems(context.Background(), s.repoFactory, allocAccountID,
		[]string{"it_a", "it_b", "it_c"})

	s.Require().NotNil(err)
	s.Equal([]string{"it_a"}, s.allocated)
}
