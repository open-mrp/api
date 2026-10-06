package service

import (
	"context"
	"testing"

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

const productLineAccessTestAccount = "ac_seller"

type ProductLineAccessSvcTestSuite struct {
	suite.Suite
	customerSvc domain.CustomerProductLineAccessSvc
	groupSvc    domain.AccountGroupProductLineAccessSvc

	customerRepo *repositorymock.MockCustomerProductLineAccessRepo
	groupRepo    *repositorymock.MockAccountGroupProductLineAccessRepo

	ctrl *gomock.Controller
}

func (suite *ProductLineAccessSvcTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.customerRepo = repositorymock.NewMockCustomerProductLineAccessRepo(suite.ctrl)
	suite.groupRepo = repositorymock.NewMockAccountGroupProductLineAccessRepo(suite.ctrl)
	repoFactory := factorymock.NewMockRepoFactory(suite.ctrl)
	repoFactory.EXPECT().NewCustomerProductLineAccessRepo().Return(suite.customerRepo).AnyTimes()
	repoFactory.EXPECT().NewAccountGroupProductLineAccessRepo().Return(suite.groupRepo).AnyTimes()
	repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	idempotencyMed := mediatormock.NewMockIdempotencyMed(suite.ctrl)
	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_access", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil).AnyTimes()
	idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_access", gomock.Any()).Return(nil).AnyTimes()
	idempotencyMed.EXPECT().CacheErrorResponse(gomock.Any(), "idk_access", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr }).AnyTimes()
	mediatorFactory := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: idempotencyMed}).AnyTimes()

	txManager := &stubTxManager{factory: repoFactory}
	suite.customerSvc = NewCustomerProductLineAccessSvc(&CustomerProductLineAccessSvcConfig{
		Repos: repoFactory, MediatorFactory: mediatorFactory, TxManager: txManager,
	})
	suite.groupSvc = NewAccountGroupProductLineAccessSvc(&AccountGroupProductLineAccessSvcConfig{
		Repos: repoFactory, MediatorFactory: mediatorFactory, TxManager: txManager,
	})
}

func (suite *ProductLineAccessSvcTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestProductLineAccessSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ProductLineAccessSvcTestSuite))
}

func productLineAccessCtx() context.Context {
	accountID := productLineAccessTestAccount
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_staff",
			AccountID:    &accountID,
			Permissions: map[string]bool{
				"relevant_products:create": true,
				"relevant_products:update": true,
			},
		},
	})
}

func (suite *ProductLineAccessSvcTestSuite) TestACustomerGrantNamingALineTwiceGrantsItOnce() {
	suite.customerRepo.EXPECT().ExistsByCustomerID(gomock.Any(), productLineAccessTestAccount, "ac_buyer").Return(false, nil)
	suite.customerRepo.EXPECT().Create(gomock.Any(), domain.CreateCustomerProductLineAccessParams{
		AccountID: productLineAccessTestAccount, CustomerID: "ac_buyer", ProductLineIDs: []string{"pl_a", "pl_b"},
	}).Return(&domain.CustomerProductLineAccess{CustomerID: "ac_buyer"}, nil)

	_, apiErr := suite.customerSvc.CreateCustomerProductLineAccess(productLineAccessCtx(), domain.CreateCustomerProductLineAccessParams{
		CustomerID: "ac_buyer", ProductLineIDs: []string{"pl_a", "pl_b", "pl_a"},
	})

	suite.Nil(apiErr)
}

func (suite *ProductLineAccessSvcTestSuite) TestACustomerEditNamingALineTwiceGrantsItOnce() {
	suite.customerRepo.EXPECT().Get(gomock.Any(), productLineAccessTestAccount, "ac_buyer").Return(&domain.CustomerProductLineAccess{CustomerID: "ac_buyer"}, nil)
	suite.customerRepo.EXPECT().Update(gomock.Any(), domain.UpdateCustomerProductLineAccessParams{
		AccountID: productLineAccessTestAccount, CustomerID: "ac_buyer", ProductLineIDs: []string{"pl_b"},
	}).Return(&domain.CustomerProductLineAccess{CustomerID: "ac_buyer"}, nil)

	_, apiErr := suite.customerSvc.UpdateCustomerProductLineAccess(productLineAccessCtx(), domain.UpdateCustomerProductLineAccessParams{
		CustomerID: "ac_buyer", ProductLineIDs: []string{"pl_b", "pl_b"},
	})

	suite.Nil(apiErr)
}

func (suite *ProductLineAccessSvcTestSuite) TestAGroupGrantNamingALineTwiceGrantsItOnce() {
	suite.groupRepo.EXPECT().ExistsByAccountGroupID(gomock.Any(), "ag_retail").Return(false, nil)
	suite.groupRepo.EXPECT().Create(gomock.Any(), domain.CreateAccountGroupProductLineAccessParams{
		AccountID: productLineAccessTestAccount, AccountGroupID: "ag_retail", ProductLineIDs: []string{"pl_a"},
	}).Return(&domain.AccountGroupProductLineAccess{AccountGroupID: "ag_retail"}, nil)

	_, apiErr := suite.groupSvc.CreateAccountGroupProductLineAccess(productLineAccessCtx(), domain.CreateAccountGroupProductLineAccessParams{
		AccountGroupID: "ag_retail", ProductLineIDs: []string{"pl_a", "pl_a"},
	})

	suite.Nil(apiErr)
}

func (suite *ProductLineAccessSvcTestSuite) TestAGroupEditNamingALineTwiceGrantsItOnce() {
	suite.groupRepo.EXPECT().Get(gomock.Any(), productLineAccessTestAccount, "ag_retail").Return(&domain.AccountGroupProductLineAccess{AccountGroupID: "ag_retail"}, nil)
	suite.groupRepo.EXPECT().Update(gomock.Any(), domain.UpdateAccountGroupProductLineAccessParams{
		AccountID: productLineAccessTestAccount, AccountGroupID: "ag_retail", ProductLineIDs: []string{"pl_b", "pl_a"},
	}).Return(&domain.AccountGroupProductLineAccess{AccountGroupID: "ag_retail"}, nil)

	_, apiErr := suite.groupSvc.UpdateAccountGroupProductLineAccess(productLineAccessCtx(), domain.UpdateAccountGroupProductLineAccessParams{
		AccountGroupID: "ag_retail", ProductLineIDs: []string{"pl_b", "pl_a", "pl_b"},
	})

	suite.Nil(apiErr)
}
