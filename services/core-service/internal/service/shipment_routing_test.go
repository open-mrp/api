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
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// Covers the routing a shipment update may point at: a carrier or service level outside the account
// is refused before anything is written, and the routing the shipment already has is not re-checked.
type ShipmentRoutingTestSuite struct {
	suite.Suite
	svc domain.ShipmentSvc

	shipmentRepo     *repositorymock.MockShipmentRepo
	shippingCaseRepo *repositorymock.MockShippingCaseRepo
	carrierRepo      *repositorymock.MockCarrierRepo
	serviceLevelRepo *repositorymock.MockServiceLevelRepo
	idempotencyMed   *mediatormock.MockIdempotencyMed

	ctrl *gomock.Controller
}

func (suite *ShipmentRoutingTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.shipmentRepo = repositorymock.NewMockShipmentRepo(suite.ctrl)
	suite.shippingCaseRepo = repositorymock.NewMockShippingCaseRepo(suite.ctrl)
	suite.carrierRepo = repositorymock.NewMockCarrierRepo(suite.ctrl)
	suite.serviceLevelRepo = repositorymock.NewMockServiceLevelRepo(suite.ctrl)
	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)

	repos := factorymock.NewMockRepoFactory(suite.ctrl)
	repos.EXPECT().NewShipmentRepo().Return(suite.shipmentRepo).AnyTimes()
	repos.EXPECT().NewShippingCaseRepo().Return(suite.shippingCaseRepo).AnyTimes()
	repos.EXPECT().NewCarrierRepo().Return(suite.carrierRepo).AnyTimes()
	repos.EXPECT().NewServiceLevelRepo().Return(suite.serviceLevelRepo).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	mediators := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: suite.idempotencyMed}).AnyTimes()

	suite.svc = NewShipmentSvc(&ShipmentSvcConfig{
		Repos:           repos,
		MediatorFactory: mediators,
		TxManager:       &stubTxManager{factory: repos},
		DispatchLeases:  newMemLeases(),
	})

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_routing", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil).
		AnyTimes()
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), "idk_routing", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr }).
		AnyTimes()
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_routing", gomock.Any()).Return(nil).AnyTimes()
}

func (suite *ShipmentRoutingTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestShipmentRoutingTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ShipmentRoutingTestSuite))
}

func shipmentWriterCtx(accountID string, role constants.RoleType) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &accountID,
			RoleType:     new(string(role)),
			Permissions:  map[string]bool{"shipments:update": true},
		},
	})
}

func (suite *ShipmentRoutingTestSuite) expectShipment(shipment *domain.Shipment) {
	suite.shipmentRepo.EXPECT().
		Get(gomock.Any(), domain.GetShipmentParams{AccountID: "ac_seller", ShipmentID: "sh_1"}).
		Return(shipment, nil)
}

func (suite *ShipmentRoutingTestSuite) TestUpdate_RefusesACarrierOutsideTheAccount() {
	suite.expectShipment(&domain.Shipment{ID: "sh_1", CarrierID: "cr_own"})
	suite.carrierRepo.EXPECT().
		Get(gomock.Any(), domain.GetCarrierParams{AccountID: "ac_seller", CarrierID: "cr_foreign"}).
		Return(nil, apierror.NewResourceNotFoundError("Resource not found."))

	_, apiErr := suite.svc.UpdateShipment(shipmentWriterCtx("ac_seller", constants.RoleTypeAdmin), domain.UpdateShipmentParams{
		ShipmentID: "sh_1",
		CarrierID:  new("cr_foreign"),
	})

	suite.Require().NotNil(apiErr)
	suite.True(apierror.IsNotFound(apiErr))
	suite.Equal("carrier_id", apiErr.Param)
}

func (suite *ShipmentRoutingTestSuite) TestUpdate_RefusesAServiceLevelOutsideTheAccount() {
	suite.expectShipment(&domain.Shipment{ID: "sh_1", CarrierID: "cr_own"})
	suite.serviceLevelRepo.EXPECT().
		Get(gomock.Any(), "ac_seller", "crop_foreign").
		Return(nil, apierror.NewResourceNotFoundError("Resource not found."))

	_, apiErr := suite.svc.UpdateShipment(shipmentWriterCtx("ac_seller", constants.RoleTypeAdmin), domain.UpdateShipmentParams{
		ShipmentID:     "sh_1",
		ServiceLevelID: field.Set("crop_foreign"),
	})

	suite.Require().NotNil(apiErr)
	suite.True(apierror.IsNotFound(apiErr))
	suite.Equal("service_level_id", apiErr.Param)
}

func (suite *ShipmentRoutingTestSuite) TestUpdate_KeepsTheCurrentRoutingWithoutLookingItUp() {
	suite.expectShipment(&domain.Shipment{ID: "sh_1", CarrierID: "cr_own", ServiceLevelID: new("crop_own")})
	suite.shippingCaseRepo.EXPECT().RepointToCarrier(gomock.Any(), "ac_seller", "sh_1", "cr_own").Return(nil)
	suite.shipmentRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&domain.Shipment{ID: "sh_1", CarrierID: "cr_own"}, nil)

	_, apiErr := suite.svc.UpdateShipment(shipmentWriterCtx("ac_seller", constants.RoleTypeAdmin), domain.UpdateShipmentParams{
		ShipmentID:     "sh_1",
		Note:           new("Leave at dock 4"),
		CarrierID:      new("cr_own"),
		ServiceLevelID: field.Set("crop_own"),
	})

	suite.Nil(apiErr)
}

func (suite *ShipmentRoutingTestSuite) TestAdminTracking_RefusesACarrierOutsideTheAccount() {
	suite.expectShipment(&domain.Shipment{ID: "sh_1", CarrierID: "cr_own", ShippedAt: new(time.Now())})
	suite.carrierRepo.EXPECT().
		Get(gomock.Any(), domain.GetCarrierParams{AccountID: "ac_seller", CarrierID: "cr_foreign"}).
		Return(nil, apierror.NewResourceNotFoundError("Resource not found."))

	_, apiErr := suite.svc.AdminUpdateShipmentTracking(shipmentWriterCtx("ac_seller", constants.RoleTypeAdmin), domain.AdminUpdateShipmentTrackingParams{
		ShipmentID: "sh_1",
		CarrierID:  new("cr_foreign"),
	})

	suite.Require().NotNil(apiErr)
	suite.True(apierror.IsNotFound(apiErr))
	suite.Equal("carrier_id", apiErr.Param)
}

func (suite *ShipmentRoutingTestSuite) TestUpdate_RefusesAServiceLevelOffTheHeldCarrier() {
	suite.expectShipment(&domain.Shipment{ID: "sh_1", CarrierID: "cr_own", ServiceLevelID: new("crop_own")})
	suite.serviceLevelRepo.EXPECT().Get(gomock.Any(), "ac_seller", "crop_other").Return(&domain.ServiceLevel{ID: "crop_other"}, nil)
	suite.serviceLevelRepo.EXPECT().IsInCarrier(gomock.Any(), "crop_other", "cr_own").Return(false, nil)

	_, apiErr := suite.svc.UpdateShipment(shipmentWriterCtx("ac_seller", constants.RoleTypeAdmin), domain.UpdateShipmentParams{
		ShipmentID:     "sh_1",
		ServiceLevelID: field.Set("crop_other"),
	})

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
	suite.Equal("service_level_id", apiErr.Param)
}

func (suite *ShipmentRoutingTestSuite) TestUpdate_RefusesACarrierTheHeldServiceLevelIsNotOn() {
	suite.expectShipment(&domain.Shipment{ID: "sh_1", CarrierID: "cr_own", ServiceLevelID: new("crop_own")})
	suite.carrierRepo.EXPECT().
		Get(gomock.Any(), domain.GetCarrierParams{AccountID: "ac_seller", CarrierID: "cr_new"}).
		Return(&domain.Carrier{ID: "cr_new"}, nil)
	suite.serviceLevelRepo.EXPECT().IsInCarrier(gomock.Any(), "crop_own", "cr_new").Return(false, nil)

	_, apiErr := suite.svc.UpdateShipment(shipmentWriterCtx("ac_seller", constants.RoleTypeAdmin), domain.UpdateShipmentParams{
		ShipmentID: "sh_1",
		CarrierID:  new("cr_new"),
	})

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
	suite.Equal("service_level_id", apiErr.Param)
}

func (suite *ShipmentRoutingTestSuite) TestAdminTracking_RefusesACarrierTheHeldServiceLevelIsNotOn() {
	suite.expectShipment(&domain.Shipment{ID: "sh_1", CarrierID: "cr_own", ServiceLevelID: new("crop_own"), ShippedAt: new(time.Now())})
	suite.carrierRepo.EXPECT().
		Get(gomock.Any(), domain.GetCarrierParams{AccountID: "ac_seller", CarrierID: "cr_new"}).
		Return(&domain.Carrier{ID: "cr_new"}, nil)
	suite.serviceLevelRepo.EXPECT().IsInCarrier(gomock.Any(), "crop_own", "cr_new").Return(false, nil)

	_, apiErr := suite.svc.AdminUpdateShipmentTracking(shipmentWriterCtx("ac_seller", constants.RoleTypeAdmin), domain.AdminUpdateShipmentTrackingParams{
		ShipmentID: "sh_1",
		CarrierID:  new("cr_new"),
	})

	suite.Require().NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
	suite.Equal("service_level_id", apiErr.Param)
}
