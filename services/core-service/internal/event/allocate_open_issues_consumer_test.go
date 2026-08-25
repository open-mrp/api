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
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

const (
	pageAccountID = "acct_test"
	pageItemID    = "it_a"
)

type continuationOutboxRepo struct {
	enqueued *[]domain.AllocateOpenIssuesEvent
}

func (r continuationOutboxRepo) Create(_ context.Context, input messaging.OutboxMessageInput) (int64, error) {
	if input.MessageType == string(contracts.CoreCmdAllocateOpenIssues) {
		var evt domain.AllocateOpenIssuesEvent
		if err := json.Unmarshal(input.Payload.Data, &evt); err != nil {
			return 0, err
		}
		*r.enqueued = append(*r.enqueued, evt)
	}
	return 0, nil
}

type AllocateOpenIssuesConsumerTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	repoFactory     *factorymock.MockRepoFactory
	reservationRepo *repositorymock.MockInventoryReservationRepo
	continuations   []domain.AllocateOpenIssuesEvent
}

func (s *AllocateOpenIssuesConsumerTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.continuations = nil

	s.reservationRepo = repositorymock.NewMockInventoryReservationRepo(s.ctrl)
	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)
	s.repoFactory.EXPECT().NewInventoryReservationRepo().Return(s.reservationRepo).AnyTimes()
	s.repoFactory.EXPECT().NewOutboxRepo().
		Return(continuationOutboxRepo{enqueued: &s.continuations}).AnyTimes()
}

func (s *AllocateOpenIssuesConsumerTestSuite) TearDownTest() { s.ctrl.Finish() }

func TestAllocateOpenIssuesConsumerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AllocateOpenIssuesConsumerTestSuite))
}

func (s *AllocateOpenIssuesConsumerTestSuite) TestFullPageEnqueuesContinuationFromCursor() {
	lastAt := time.Date(2026, 3, 16, 22, 18, 40, 0, time.UTC)
	s.reservationRepo.EXPECT().
		AllocateOpenIssuesForItemPage(gomock.Any(), pageAccountID, pageItemID, time.Time{}, "", int32(3)).
		Return(lastAt, "ii_last", 3, nil)

	err := allocateOpenIssuesPage(context.Background(), s.repoFactory, pageAccountID, pageItemID, time.Time{}, "", 3)

	s.Require().Nil(err)
	s.Require().Len(s.continuations, 1)
	s.Equal("ii_last", s.continuations[0].AfterID)
	s.True(s.continuations[0].AfterCreatedAt.Equal(lastAt))
}

func (s *AllocateOpenIssuesConsumerTestSuite) TestShortPageStopsPaging() {
	prevAt := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	s.reservationRepo.EXPECT().
		AllocateOpenIssuesForItemPage(gomock.Any(), pageAccountID, pageItemID, prevAt, "ii_5", int32(3)).
		Return(time.Date(2026, 3, 16, 11, 0, 0, 0, time.UTC), "ii_7", 2, nil)

	err := allocateOpenIssuesPage(context.Background(), s.repoFactory, pageAccountID, pageItemID, prevAt, "ii_5", 3)

	s.Require().Nil(err)
	s.Empty(s.continuations)
}

func (s *AllocateOpenIssuesConsumerTestSuite) TestEmptyPageStopsPaging() {
	prevAt := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	s.reservationRepo.EXPECT().
		AllocateOpenIssuesForItemPage(gomock.Any(), pageAccountID, pageItemID, prevAt, "ii_9", int32(3)).
		Return(prevAt, "ii_9", 0, nil)

	err := allocateOpenIssuesPage(context.Background(), s.repoFactory, pageAccountID, pageItemID, prevAt, "ii_9", 3)

	s.Require().Nil(err)
	s.Empty(s.continuations)
}

func (s *AllocateOpenIssuesConsumerTestSuite) TestPageFailureStopsAndReports() {
	s.reservationRepo.EXPECT().
		AllocateOpenIssuesForItemPage(gomock.Any(), pageAccountID, pageItemID, time.Time{}, "", int32(3)).
		Return(time.Time{}, "", 0, apierror.NewInternalError(nil, "boom"))

	err := allocateOpenIssuesPage(context.Background(), s.repoFactory, pageAccountID, pageItemID, time.Time{}, "", 3)

	s.Require().NotNil(err)
	s.Empty(s.continuations)
}
