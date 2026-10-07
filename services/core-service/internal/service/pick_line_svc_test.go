package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
)

const pickLineTestLineID = "pkln_picktest"

type PickLineSvcTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	pickRepo       *repositorymock.MockPickRepo
	pickLineRepo   *repositorymock.MockPickLineRepo
	idempotencyMed *mediatormock.MockIdempotencyMed
	svc            domain.PickLineSvc
}

func TestPickLineSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PickLineSvcTestSuite))
}

func (suite *PickLineSvcTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.pickRepo = repositorymock.NewMockPickRepo(suite.ctrl)
	suite.pickLineRepo = repositorymock.NewMockPickLineRepo(suite.ctrl)
	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)

	repos := factorymock.NewMockRepoFactory(suite.ctrl)
	repos.EXPECT().NewPickRepo().Return(suite.pickRepo).AnyTimes()
	repos.EXPECT().NewPickLineRepo().Return(suite.pickLineRepo).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	mediators := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: suite.idempotencyMed}).AnyTimes()

	suite.svc = NewPickLineSvc(&PickLineSvcConfig{
		Repos:           repos,
		MediatorFactory: mediators,
		TxManager:       &stubTxManager{factory: repos},
	})

	// Every case here reaches the transaction for a line that belongs to the caller's pick.
	suite.pickRepo.EXPECT().IsInAccount(gomock.Any(), pickTestAccountID, pickTestPickID).Return(true, nil)
	suite.pickLineRepo.EXPECT().IsInPick(gomock.Any(), pickLineTestLineID, pickTestPickID).Return(true, nil)
	suite.pickRepo.EXPECT().GetSalesOrderForPick(gomock.Any(), pickTestAccountID, pickTestPickID).
		Return(&domain.PickSalesOrder{ID: "so_1"}, nil)
	suite.idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_line", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
}

func (suite *PickLineSvcTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *PickLineSvcTestSuite) expectErrorCached() {
	suite.idempotencyMed.EXPECT().CacheErrorResponse(gomock.Any(), "idk_line", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr })
}

func (suite *PickLineSvcTestSuite) expectSuccessCached() {
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_line", gomock.Any()).Return(nil)
}

// A packed line's quantity is what its shipment carries; editing it would desync the two.
func (suite *PickLineSvcTestSuite) TestUpdate_RefusesAPackedLine() {
	suite.pickLineRepo.EXPECT().LockUnpacked(gomock.Any(), pickLineTestLineID).Return(false, nil)
	suite.expectErrorCached()

	value := "3"
	line, apiErr := suite.svc.UpdatePickLine(pickTestCtx(), domain.UpdatePickLineParams{
		PickID: pickTestPickID, PickLineID: pickLineTestLineID, QuantityValue: &value,
	})

	suite.Nil(line)
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *PickLineSvcTestSuite) TestUpdate_LocksTheLineBeforeWriting() {
	value := "3"
	gomock.InOrder(
		suite.pickLineRepo.EXPECT().LockUnpacked(gomock.Any(), pickLineTestLineID).Return(true, nil),
		suite.pickLineRepo.EXPECT().Get(gomock.Any(), pickLineTestLineID).Return(&domain.PickLine{ID: pickLineTestLineID, QuantityValue: "0"}, nil),
		suite.pickLineRepo.EXPECT().UpdateQuantity(gomock.Any(), pickLineTestLineID, &value, nil).Return(nil),
		suite.pickLineRepo.EXPECT().Get(gomock.Any(), pickLineTestLineID).Return(&domain.PickLine{ID: pickLineTestLineID, QuantityValue: "3"}, nil),
	)
	suite.expectSuccessCached()

	line, apiErr := suite.svc.UpdatePickLine(pickTestCtx(), domain.UpdatePickLineParams{
		PickID: pickTestPickID, PickLineID: pickLineTestLineID, QuantityValue: &value,
	})

	suite.Require().Nil(apiErr)
	suite.Equal("3", line.QuantityValue)
}

// The packed check runs under the line lock, so a pack committing in between is seen, not raced.
func (suite *PickLineSvcTestSuite) TestVoid_RefusesAPackedLine() {
	suite.pickLineRepo.EXPECT().LockUnpacked(gomock.Any(), pickLineTestLineID).Return(false, nil)
	suite.expectErrorCached()

	line, apiErr := suite.svc.VoidPickLine(pickTestCtx(), pickTestPickID, pickLineTestLineID)

	suite.Nil(line)
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func (suite *PickLineSvcTestSuite) TestVoid_ZeroesAnOpenLine() {
	gomock.InOrder(
		suite.pickLineRepo.EXPECT().LockUnpacked(gomock.Any(), pickLineTestLineID).Return(true, nil),
		suite.pickLineRepo.EXPECT().Get(gomock.Any(), pickLineTestLineID).Return(&domain.PickLine{ID: pickLineTestLineID, QuantityValue: "4"}, nil),
		suite.pickLineRepo.EXPECT().VoidLine(gomock.Any(), pickLineTestLineID).Return(nil),
		suite.pickLineRepo.EXPECT().Get(gomock.Any(), pickLineTestLineID).Return(&domain.PickLine{ID: pickLineTestLineID, QuantityValue: "0"}, nil),
	)
	suite.expectSuccessCached()

	line, apiErr := suite.svc.VoidPickLine(pickTestCtx(), pickTestPickID, pickLineTestLineID)

	suite.Require().Nil(apiErr)
	suite.Equal("0", line.QuantityValue)
}

func (suite *PickLineSvcTestSuite) TestPick_LocksThePickBeforeTouchingTheLine() {
	gomock.InOrder(
		suite.pickRepo.EXPECT().Lock(gomock.Any(), pickTestAccountID, pickTestPickID).Return(false, nil),
		suite.pickLineRepo.EXPECT().Get(gomock.Any(), pickLineTestLineID).Return(&domain.PickLine{ID: pickLineTestLineID, QuantityValue: "0"}, nil),
		suite.pickLineRepo.EXPECT().PickRemainingQuantity(gomock.Any(), pickLineTestLineID).Return(nil),
		suite.pickLineRepo.EXPECT().Get(gomock.Any(), pickLineTestLineID).Return(&domain.PickLine{ID: pickLineTestLineID, QuantityValue: "4"}, nil),
	)
	suite.expectSuccessCached()

	line, apiErr := suite.svc.PickPickLine(pickTestCtx(), pickTestPickID, pickLineTestLineID)

	suite.Require().Nil(apiErr)
	suite.Equal("4", line.QuantityValue)
}
