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

// ShipmentSvcGetTestSuite covers customer-safe GetShipment authorization: assigned
// actors with counterparty access may read their own shipments; other buyers'
// shipments on the same vendor account are not found.
type ShipmentSvcGetTestSuite struct {
	suite.Suite
	svc domain.ShipmentSvc

	shipmentRepo    *repositorymock.MockShipmentRepo
	repoFactory     *factorymock.MockRepoFactory
	mediatorFactory *factorymock.MockMediatorFactory
	readAccessMed   *mediatormock.MockReadAccessMed

	ctrl *gomock.Controller
}

func (suite *ShipmentSvcGetTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.shipmentRepo = repositorymock.NewMockShipmentRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewShipmentRepo().Return(suite.shipmentRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.readAccessMed = mediatormock.NewMockReadAccessMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		ReadAccess: suite.readAccessMed,
	}).AnyTimes()

	suite.svc = NewShipmentSvc(&ShipmentSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *ShipmentSvcGetTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestShipmentSvcGetTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ShipmentSvcGetTestSuite))
}

func shipmentCustomerCtx(targetAccountID, customerAccountID string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeCustomer,
			ID:           "usr_customer",
			AccountID:    &customerAccountID,
			Permissions:  map[string]bool{},
		},
	})
}

func shipmentInternalCtx(accountID string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &accountID,
			Permissions:  map[string]bool{"shipments:read": true},
		},
	})
}

func (suite *ShipmentSvcGetTestSuite) TestGetShipment_InternalActor() {
	ctx := shipmentInternalCtx("ac_vendor")

	suite.shipmentRepo.EXPECT().
		Get(gomock.Any(), domain.GetShipmentParams{AccountID: "ac_vendor", ShipmentID: "sh_1"}).
		Return(&domain.Shipment{ID: "sh_1", CustomerID: "ac_other"}, nil).
		Times(1)

	result, apiErr := suite.svc.GetShipment(ctx, domain.GetShipmentParams{ShipmentID: "sh_1"})
	suite.Require().Nil(apiErr)
	suite.Require().NotNil(result)
	suite.Equal("sh_1", result.ID)
}

func (suite *ShipmentSvcGetTestSuite) TestGetShipment_CustomerActor_OwnShipment() {
	ctx := shipmentCustomerCtx("ac_vendor", "ac_customer")

	suite.readAccessMed.EXPECT().
		CheckCounterpartyReadAccess(gomock.Any(), "ac_customer", "ac_vendor").
		Return(nil).
		Times(1)

	suite.shipmentRepo.EXPECT().
		Get(gomock.Any(), domain.GetShipmentParams{AccountID: "ac_vendor", ShipmentID: "sh_1"}).
		Return(&domain.Shipment{ID: "sh_1", Number: "SHP-001", CustomerID: "ac_customer"}, nil).
		Times(1)

	result, apiErr := suite.svc.GetShipment(ctx, domain.GetShipmentParams{ShipmentID: "sh_1"})
	suite.Require().Nil(apiErr)
	suite.Require().NotNil(result)
	suite.Equal("sh_1", result.ID)
	suite.Equal("SHP-001", result.Number)
}

func (suite *ShipmentSvcGetTestSuite) TestGetShipment_CustomerActor_OtherBuyerNotFound() {
	ctx := shipmentCustomerCtx("ac_vendor", "ac_customer")

	suite.readAccessMed.EXPECT().
		CheckCounterpartyReadAccess(gomock.Any(), "ac_customer", "ac_vendor").
		Return(nil).
		Times(1)

	suite.shipmentRepo.EXPECT().
		Get(gomock.Any(), domain.GetShipmentParams{AccountID: "ac_vendor", ShipmentID: "sh_other"}).
		Return(&domain.Shipment{ID: "sh_other", CustomerID: "ac_other_customer"}, nil).
		Times(1)

	result, apiErr := suite.svc.GetShipment(ctx, domain.GetShipmentParams{ShipmentID: "sh_other"})
	suite.Nil(result)
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceNotFound, apiErr.Code)
}

func (suite *ShipmentSvcGetTestSuite) TestGetShipment_CustomerActor_NoRelation() {
	ctx := shipmentCustomerCtx("ac_vendor", "ac_customer")

	suite.readAccessMed.EXPECT().
		CheckCounterpartyReadAccess(gomock.Any(), "ac_customer", "ac_vendor").
		Return(apierror.NewAuthorizationError("You are not authorized to access this resource.")).
		Times(1)

	result, apiErr := suite.svc.GetShipment(ctx, domain.GetShipmentParams{ShipmentID: "sh_1"})
	suite.Nil(result)
	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}
