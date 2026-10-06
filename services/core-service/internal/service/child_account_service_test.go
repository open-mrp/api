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

const childAccountTestSeller = "ac_seller"

type ChildAccountSvcTestSuite struct {
	suite.Suite
	svc domain.ChildAccountSvc

	relationRepo  *repositorymock.MockAccountRelationRepo
	customerRepo  *repositorymock.MockCustomerRepo
	editAccessMed *mediatormock.MockEditAccessMed

	ctrl *gomock.Controller
}

func (suite *ChildAccountSvcTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.relationRepo = repositorymock.NewMockAccountRelationRepo(suite.ctrl)
	suite.customerRepo = repositorymock.NewMockCustomerRepo(suite.ctrl)
	repoFactory := factorymock.NewMockRepoFactory(suite.ctrl)
	repoFactory.EXPECT().NewAccountRelationRepo().Return(suite.relationRepo).AnyTimes()
	repoFactory.EXPECT().NewCustomerRepo().Return(suite.customerRepo).AnyTimes()
	repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.editAccessMed = mediatormock.NewMockEditAccessMed(suite.ctrl)
	suite.editAccessMed.EXPECT().CheckEditAccess(gomock.Any(), childAccountTestSeller, gomock.Any()).Return(nil).AnyTimes()
	mediatorFactory := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{EditAccess: suite.editAccessMed}).AnyTimes()

	suite.svc = NewChildAccountSvc(&ChildAccountSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       &stubTxManager{factory: repoFactory},
	})
}

func (suite *ChildAccountSvcTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestChildAccountSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ChildAccountSvcTestSuite))
}

// sellerStaffOn is the seller's staff working on one of its customers' pages.
func sellerStaffOn(parentAccountID string) context.Context {
	seller := childAccountTestSeller
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: parentAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_staff",
			AccountID:    &seller,
			Permissions:  map[string]bool{"customers:update": true},
		},
	})
}

// customers registers each account as one of the seller's customers, its relation named after it.
func (suite *ChildAccountSvcTestSuite) customers(accountIDs ...string) {
	for _, accountID := range accountIDs {
		suite.customerRepo.EXPECT().GetRelationID(gomock.Any(), childAccountTestSeller, accountID).Return("rel_"+accountID, nil).AnyTimes()
	}
}

func (suite *ChildAccountSvcTestSuite) parentOf(relationID string, parentRelationID *string) {
	suite.relationRepo.EXPECT().GetParentRelationID(gomock.Any(), relationID).Return(parentRelationID, nil).AnyTimes()
}

func (suite *ChildAccountSvcTestSuite) TestAnAccountCannotBeItsOwnParent() {
	suite.customers("ac_a")

	_, apiErr := suite.svc.AddChildAccount(sellerStaffOn("ac_a"), "ac_a")

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
}

func (suite *ChildAccountSvcTestSuite) TestAnAccountCannotBeLinkedBeneathItsGrandchild() {
	suite.customers("ac_a", "ac_c")
	suite.parentOf("rel_ac_c", new("rel_ac_b"))
	suite.parentOf("rel_ac_b", new("rel_ac_a"))

	_, apiErr := suite.svc.AddChildAccount(sellerStaffOn("ac_c"), "ac_a")

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
}

func (suite *ChildAccountSvcTestSuite) TestAWalkThroughCorruptDataStops() {
	suite.customers("ac_a", "ac_x")
	suite.parentOf("rel_ac_x", new("rel_ac_y"))
	suite.parentOf("rel_ac_y", new("rel_ac_x"))

	_, apiErr := suite.svc.AddChildAccount(sellerStaffOn("ac_x"), "ac_a")

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
}

func (suite *ChildAccountSvcTestSuite) TestTheChildMustBeOneOfTheSellersCustomers() {
	suite.customers("ac_parent")
	suite.customerRepo.EXPECT().GetRelationID(gomock.Any(), childAccountTestSeller, "ac_supplier").
		Return("", apierror.NewResourceNotFoundError("Resource not found."))

	_, apiErr := suite.svc.AddChildAccount(sellerStaffOn("ac_parent"), "ac_supplier")

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
	suite.Equal("Child account not found.", apiErr.PublicMessage)
}

func (suite *ChildAccountSvcTestSuite) TestLinksAChildBeneathAnotherCustomer() {
	suite.customers("ac_parent", "ac_child")
	suite.parentOf("rel_ac_parent", new("rel_ac_head"))
	suite.parentOf("rel_ac_head", nil)
	suite.relationRepo.EXPECT().SetParentRelation(gomock.Any(), childAccountTestSeller, "rel_ac_child", "rel_ac_parent").Return(nil)
	suite.relationRepo.EXPECT().GetChildAccountDetail(gomock.Any(), childAccountTestSeller, "ac_child").
		Return(&domain.ChildAccount{RelationID: "rel_ac_child", AccountID: "ac_child"}, nil)

	got, apiErr := suite.svc.AddChildAccount(sellerStaffOn("ac_parent"), "ac_child")

	suite.Require().Nil(apiErr)
	suite.Equal("rel_ac_child", got.RelationID)
}
