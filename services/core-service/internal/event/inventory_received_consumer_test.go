package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
)

const allocAccountID = "acct_test"

type InventoryReceivedConsumerTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	repoFactory     *factorymock.MockRepoFactory
	reservationRepo *repositorymock.MockInventoryReservationRepo
	allocated       []string
}

func (s *InventoryReceivedConsumerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.allocated = nil

	s.reservationRepo = repositorymock.NewMockInventoryReservationRepo(s.ctrl)
	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)
	s.repoFactory.EXPECT().NewInventoryReservationRepo().Return(s.reservationRepo).AnyTimes()
}

func (s *InventoryReceivedConsumerTestSuite) TearDownTest() { s.ctrl.Finish() }

func TestInventoryReceivedConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InventoryReceivedConsumerTestSuite))
}

func (s *InventoryReceivedConsumerTestSuite) expectAllocation() {
	s.reservationRepo.EXPECT().
		AllocateOpenIssuesForItem(gomock.Any(), allocAccountID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, itemID string) *apierror.APIError {
			s.allocated = append(s.allocated, itemID)
			return nil
		}).AnyTimes()
}

// Every item whose stock moved is offered to the demand waiting on it, in the order given.
func (s *InventoryReceivedConsumerTestSuite) TestAllocatesEveryItem() {
	s.expectAllocation()

	err := allocateForItems(context.Background(), s.repoFactory, allocAccountID,
		[]string{"it_a", "it_b", "it_c"})

	s.Require().Nil(err)
	s.Equal([]string{"it_a", "it_b", "it_c"}, s.allocated)
}

// One cause routinely names an item twice — a step consuming a material in two places. Allocating
// twice for one arrival walks the same open issues again for nothing.
func (s *InventoryReceivedConsumerTestSuite) TestAllocatesEachItemOnce() {
	s.expectAllocation()

	err := allocateForItems(context.Background(), s.repoFactory, allocAccountID,
		[]string{"it_a", "it_b", "it_a", "it_a"})

	s.Require().Nil(err)
	s.Equal([]string{"it_a", "it_b"}, s.allocated)
}

func (s *InventoryReceivedConsumerTestSuite) TestIgnoresEmptyItemIDs() {
	s.expectAllocation()

	err := allocateForItems(context.Background(), s.repoFactory, allocAccountID,
		[]string{"", "it_a", ""})

	s.Require().Nil(err)
	s.Equal([]string{"it_a"}, s.allocated)
}

func (s *InventoryReceivedConsumerTestSuite) TestNoItemsIsNoWork() {
	err := allocateForItems(context.Background(), s.repoFactory, allocAccountID, nil)

	s.Require().Nil(err)
	s.Empty(s.allocated)
}

// A failure stops the run rather than continuing, so the whole transaction rolls back and the
// delivery is retried. Allocating the rest and reporting success would leave the failed item short
// with nothing recording that allocation was still owed — which is the failure mode the nightly
// sweep used to paper over.
func (s *InventoryReceivedConsumerTestSuite) TestStopsAndReportsOnFailure() {
	s.reservationRepo.EXPECT().
		AllocateOpenIssuesForItem(gomock.Any(), allocAccountID, "it_a").
		Return(nil)
	s.reservationRepo.EXPECT().
		AllocateOpenIssuesForItem(gomock.Any(), allocAccountID, "it_b").
		Return(apierror.NewInternalError(nil, "boom"))

	err := allocateForItems(context.Background(), s.repoFactory, allocAccountID,
		[]string{"it_a", "it_b", "it_c"})

	s.Require().NotNil(err)
}
