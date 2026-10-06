package service

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const salesTargetTestAccount = "ac_seller"

var (
	salesTargetTestStart = time.Date(2034, 1, 1, 0, 0, 0, 0, time.UTC)
	salesTargetTestEnd   = time.Date(2034, 3, 31, 0, 0, 0, 0, time.UTC)
)

type SalesTargetSvcTestSuite struct {
	suite.Suite
	svc domain.SalesTargetSvc

	targetRepo *repositorymock.MockSalesTargetRepo
	unitRepo   *repositorymock.MockUnitRepo

	ctrl *gomock.Controller
}

func (suite *SalesTargetSvcTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.targetRepo = repositorymock.NewMockSalesTargetRepo(suite.ctrl)
	suite.unitRepo = repositorymock.NewMockUnitRepo(suite.ctrl)
	repoFactory := factorymock.NewMockRepoFactory(suite.ctrl)
	repoFactory.EXPECT().NewSalesTargetRepo().Return(suite.targetRepo).AnyTimes()
	repoFactory.EXPECT().NewUnitRepo().Return(suite.unitRepo).AnyTimes()
	repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	mediatorFactory := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: mediatormock.NewMockIdempotencyMed(suite.ctrl)}).AnyTimes()

	suite.svc = NewSalesTargetSvc(&SalesTargetSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       &stubTxManager{factory: repoFactory},
	})

	suite.targetRepo.EXPECT().SalesRepExistsInAccount(gomock.Any(), gomock.Any(), salesTargetTestAccount).Return(true, nil).AnyTimes()
}

func (suite *SalesTargetSvcTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestSalesTargetSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SalesTargetSvcTestSuite))
}

func salesTargetCtx() context.Context {
	accountID := salesTargetTestAccount
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_staff",
			AccountID:    &accountID,
			Permissions:  map[string]bool{"sales_targets:create": true, "sales_targets:update": true},
		},
	})
}

func (suite *SalesTargetSvcTestSuite) newTarget(start, end time.Time, unitID string) domain.UpsertSalesTargetParams {
	suite.targetRepo.EXPECT().Exists(gomock.Any(), "tgt_new").Return(false, nil)
	return domain.UpsertSalesTargetParams{
		TargetID: "tgt_new", SalesRepID: "au_rep", StartDate: start, EndDate: end, AmountValue: "10", AmountUnitID: unitID,
	}
}

func (suite *SalesTargetSvcTestSuite) TestANewTargetMayNotEndBeforeItStarts() {
	params := suite.newTarget(salesTargetTestEnd, salesTargetTestStart, "un_dollar")

	_, apiErr := suite.svc.UpsertSalesTarget(salesTargetCtx(), params)

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
	suite.Equal("ends_at", apiErr.Param)
}

func (suite *SalesTargetSvcTestSuite) TestANewTargetIsNotMeasuredInAnotherAccountsUnit() {
	params := suite.newTarget(salesTargetTestStart, salesTargetTestEnd, "un_elsewhere")
	suite.unitRepo.EXPECT().Get(gomock.Any(), domain.GetUnitParams{AccountID: salesTargetTestAccount, UnitID: "un_elsewhere"}).
		Return(nil, apierror.NewResourceNotFoundError("Resource not found."))

	_, apiErr := suite.svc.UpsertSalesTarget(salesTargetCtx(), params)

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
	suite.Equal("amount_unit_id", apiErr.Param)
}

func (suite *SalesTargetSvcTestSuite) TestACreatedTargetMayNotEndBeforeItStarts() {
	_, apiErr := suite.svc.CreateSalesTarget(salesTargetCtx(), domain.CreateSalesTargetParams{
		SalesRepID: "au_rep", StartDate: salesTargetTestEnd, EndDate: salesTargetTestStart, AmountValue: "10", AmountUnitID: "un_dollar",
	})

	suite.Require().NotNil(apiErr)
	suite.Equal("ends_at", apiErr.Param)
}

func (suite *SalesTargetSvcTestSuite) TestAnotherRepsPathDoesNotReachTheTarget() {
	suite.targetRepo.EXPECT().Exists(gomock.Any(), "tgt_a").Return(true, nil)
	suite.targetRepo.EXPECT().IsInAccount(gomock.Any(), "tgt_a", salesTargetTestAccount).Return(true, nil)
	suite.targetRepo.EXPECT().Get(gomock.Any(), "tgt_a").Return(&domain.SalesTarget{ID: "tgt_a", SalesRepID: "au_rep_a", AmountID: "qty_a"}, nil)

	_, apiErr := suite.svc.UpsertSalesTarget(salesTargetCtx(), domain.UpsertSalesTargetParams{
		TargetID: "tgt_a", SalesRepID: "au_rep_b", StartDate: salesTargetTestStart, EndDate: salesTargetTestEnd, AmountValue: "999", AmountUnitID: "un_dollar",
	})

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}
