package service

import (
	"context"
	"errors"
	"testing"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// ── suite setup ──────────────────────────────────────────────────────────────

type LocationSvcTestSuite struct {
	suite.Suite
	locationSvc       domain.LocationSvc
	locationRepo      *repositorymock.MockLocationRepo
	deletedRecordRepo *repositorymock.MockDeletedRecordRepo
	repoFactory       *factorymock.MockRepoFactory
	mediatorFactory   *factorymock.MockMediatorFactory
	idempotencyMed    *mediatormock.MockIdempotencyMed
	ctrl              *gomock.Controller
}

func (suite *LocationSvcTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.locationRepo = repositorymock.NewMockLocationRepo(suite.ctrl)
	suite.deletedRecordRepo = repositorymock.NewMockDeletedRecordRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewLocationRepo().Return(suite.locationRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewDeletedRecordRepo().Return(suite.deletedRecordRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.locationSvc = NewLocationSvc(&LocationSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
		// Accept-rejection tests fail before the job is raised and write tests call
		// writeBulkUpsertLocations directly, so the factory is never exercised here; a real
		// one satisfies validate().
		JobSvcFactory: NewJobSvcFactory(),
	})
}

func (suite *LocationSvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestLocationSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(LocationSvcTestSuite))
}

// ── identity context helpers ─────────────────────────────────────────────────

func internalLocationCtx(targetAccountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_loc_test",
			AccountID:    &targetAccountID,
			RoleType:     &adminCode,
			Permissions: map[string]bool{
				"locations:read":   true,
				"locations:create": true,
				"locations:update": true,
				"locations:delete": true,
			},
		},
	})
}

func readOnlyLocationCtx(targetAccountID string) context.Context {
	customCode := string(constants.RoleTypeCustom)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_loc_test",
			AccountID:    &targetAccountID,
			RoleType:     &customCode,
			Permissions:  map[string]bool{"locations:read": true},
		},
	})
}

func createOnlyLocationCtx(targetAccountID string) context.Context {
	customCode := string(constants.RoleTypeCustom)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_loc_test",
			AccountID:    &targetAccountID,
			RoleType:     &customCode,
			Permissions: map[string]bool{
				"locations:read":   true,
				"locations:create": true,
			},
		},
	})
}

func noTargetLocationCtx() context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_loc_test",
			RoleType:     &adminCode,
			Permissions: map[string]bool{
				"locations:read":   true,
				"locations:create": true,
				"locations:update": true,
				"locations:delete": true,
			},
		},
	})
}

// ── idempotency helpers ──────────────────────────────────────────────────────

func (suite *LocationSvcTestSuite) expectLocIdempotencyStarted() {
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_loc_test",
			RecoveryPoint: string(domain.RecoveryPointStarted),
		}, nil).
		Times(1)
}

func (suite *LocationSvcTestSuite) expectLocCacheSuccess() {
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), "idk_loc_test", gomock.Any()).
		Return(nil).
		Times(1)
}

func (suite *LocationSvcTestSuite) expectLocCacheError() {
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), "idk_loc_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		}).
		Times(1)
}

// ── ListLocations ─────────────────────────────────────────────────────────────

func (suite *LocationSvcTestSuite) TestListLocations_Success() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(&domain.ListLocationsResult{Locations: []*domain.Location{{ID: "loc_1", Name: "Aisle A"}}}, nil).
		Times(1)

	result, err := suite.locationSvc.ListLocations(ctx, domain.ListLocationsParams{})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Len(result.Locations, 1)
	suite.Equal("loc_1", result.Locations[0].ID)
}

func (suite *LocationSvcTestSuite) TestListLocations_MissingIdentity() {
	result, err := suite.locationSvc.ListLocations(context.Background(), domain.ListLocationsParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *LocationSvcTestSuite) TestListLocations_MissingTargetAccount() {
	result, err := suite.locationSvc.ListLocations(noTargetLocationCtx(), domain.ListLocationsParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (suite *LocationSvcTestSuite) TestListLocations_InsufficientPermissions() {
	customCode := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_loc1"},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_loc_test",
			RoleType:     &customCode,
			Permissions:  map[string]bool{},
		},
	})

	result, err := suite.locationSvc.ListLocations(ctx, domain.ListLocationsParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *LocationSvcTestSuite) TestListLocations_ExternalActorRejected() {
	supplierCode := string(constants.RoleTypeAdmin)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_loc1"},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeSupplier,
			ID:           "usr_ext",
			RoleType:     &supplierCode,
			Permissions:  map[string]bool{"locations:read": true},
		},
	})

	result, err := suite.locationSvc.ListLocations(ctx, domain.ListLocationsParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *LocationSvcTestSuite) TestListLocations_RepoError() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("db"), "query failed")).
		Times(1)

	result, err := suite.locationSvc.ListLocations(ctx, domain.ListLocationsParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

// ── GetLocation ───────────────────────────────────────────────────────────────

func (suite *LocationSvcTestSuite) TestGetLocation_Success() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_1"}).
		Return(&domain.Location{ID: "loc_1", Name: "Aisle A"}, nil).
		Times(1)

	result, err := suite.locationSvc.GetLocation(ctx, domain.GetLocationParams{LocationID: "loc_1"})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("loc_1", result.ID)
}

func (suite *LocationSvcTestSuite) TestGetLocation_MissingIdentity() {
	result, err := suite.locationSvc.GetLocation(context.Background(), domain.GetLocationParams{LocationID: "loc_1"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *LocationSvcTestSuite) TestGetLocation_MissingTargetAccount() {
	result, err := suite.locationSvc.GetLocation(noTargetLocationCtx(), domain.GetLocationParams{LocationID: "loc_1"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (suite *LocationSvcTestSuite) TestGetLocation_InsufficientPermissions() {
	customCode := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_loc1"},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_loc_test",
			RoleType:     &customCode,
			Permissions:  map[string]bool{},
		},
	})

	result, err := suite.locationSvc.GetLocation(ctx, domain.GetLocationParams{LocationID: "loc_1"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *LocationSvcTestSuite) TestGetLocation_NotFound() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_nf"}).
		Return(nil, apierror.NewResourceNotFoundError("Location not found.")).
		Times(1)

	result, err := suite.locationSvc.GetLocation(ctx, domain.GetLocationParams{LocationID: "loc_nf"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

// ── ListLocationTypes ─────────────────────────────────────────────────────────

func (suite *LocationSvcTestSuite) TestListLocationTypes_Success() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		ListTypes(gomock.Any(), gomock.Any()).
		Return(&domain.ListLocationTypesResult{LocationTypes: []*domain.LocationType{{ID: "lt_1", Code: "bin"}}}, nil).
		Times(1)

	result, err := suite.locationSvc.ListLocationTypes(ctx, domain.ListLocationTypesParams{})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Len(result.LocationTypes, 1)
}

func (suite *LocationSvcTestSuite) TestListLocationTypes_MissingIdentity() {
	result, err := suite.locationSvc.ListLocationTypes(context.Background(), domain.ListLocationTypesParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *LocationSvcTestSuite) TestListLocationTypes_InsufficientPermissions() {
	customCode := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_loc1"},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_loc_test",
			RoleType:     &customCode,
			Permissions:  map[string]bool{},
		},
	})

	result, err := suite.locationSvc.ListLocationTypes(ctx, domain.ListLocationTypesParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

// ── GetLocationType ───────────────────────────────────────────────────────────

func (suite *LocationSvcTestSuite) TestGetLocationType_Success() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		GetType(gomock.Any(), "bin").
		Return(&domain.LocationType{ID: "lt_1", Code: "bin"}, nil).
		Times(1)

	result, err := suite.locationSvc.GetLocationType(ctx, domain.GetLocationTypeParams{Identifier: "bin"})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("bin", result.Code)
}

func (suite *LocationSvcTestSuite) TestGetLocationType_MissingIdentity() {
	result, err := suite.locationSvc.GetLocationType(context.Background(), domain.GetLocationTypeParams{Identifier: "bin"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *LocationSvcTestSuite) TestGetLocationType_NotFound() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		GetType(gomock.Any(), "nonexistent").
		Return(nil, apierror.NewResourceNotFoundError("Location type not found.")).
		Times(1)

	result, err := suite.locationSvc.GetLocationType(ctx, domain.GetLocationTypeParams{Identifier: "nonexistent"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

// ── CreateLocation ────────────────────────────────────────────────────────────

func (suite *LocationSvcTestSuite) TestCreateLocation_Success_NoParentNoChildren() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.Location{ID: "loc_new1", Name: "Bin 1"}, nil).
		Times(1)
	suite.expectLocCacheSuccess()

	result, err := suite.locationSvc.CreateLocation(ctx, domain.CreateLocationParams{
		Name:     "Bin 1",
		TypeCode: "bin",
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("loc_new1", result.ID)
}

func (suite *LocationSvcTestSuite) TestCreateLocation_Success_WithParent() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_parent").
		Return(true, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.Location{ID: "loc_new2", Name: "Bin 2", ParentID: strPtr("loc_parent")}, nil).
		Times(1)
	suite.expectLocCacheSuccess()

	result, err := suite.locationSvc.CreateLocation(ctx, domain.CreateLocationParams{
		Name:     "Bin 2",
		TypeCode: "bin",
		ParentID: strPtr("loc_parent"),
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("loc_new2", result.ID)
}

func (suite *LocationSvcTestSuite) TestCreateLocation_Success_WithChildren() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_child1").
		Return(true, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_child2").
		Return(true, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.Location{ID: "loc_new3", Name: "Aisle"}, nil).
		Times(1)
	suite.expectLocCacheSuccess()

	result, err := suite.locationSvc.CreateLocation(ctx, domain.CreateLocationParams{
		Name:     "Aisle",
		TypeCode: "aisle",
		ChildIDs: []string{"loc_child1", "loc_child2"},
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("loc_new3", result.ID)
}

func (suite *LocationSvcTestSuite) TestCreateLocation_MissingIdentity() {
	result, err := suite.locationSvc.CreateLocation(context.Background(), domain.CreateLocationParams{Name: "X"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *LocationSvcTestSuite) TestCreateLocation_MissingTargetAccount() {
	ctx := idempotencyCtx(noTargetLocationCtx())

	result, err := suite.locationSvc.CreateLocation(ctx, domain.CreateLocationParams{Name: "X"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (suite *LocationSvcTestSuite) TestCreateLocation_InsufficientPermissions() {
	ctx := readOnlyLocationCtx("ac_loc1")

	result, err := suite.locationSvc.CreateLocation(ctx, domain.CreateLocationParams{Name: "X"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *LocationSvcTestSuite) TestCreateLocation_ParentNotInAccount() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_foreign").
		Return(false, nil).
		Times(1)
	suite.expectLocCacheError()

	result, err := suite.locationSvc.CreateLocation(ctx, domain.CreateLocationParams{
		Name:     "Bin",
		TypeCode: "bin",
		ParentID: strPtr("loc_foreign"),
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("parent_id", err.Param)
}

func (suite *LocationSvcTestSuite) TestCreateLocation_ChildNotInAccount() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_foreign_child").
		Return(false, nil).
		Times(1)
	suite.expectLocCacheError()

	result, err := suite.locationSvc.CreateLocation(ctx, domain.CreateLocationParams{
		Name:     "Aisle",
		TypeCode: "aisle",
		ChildIDs: []string{"loc_foreign_child"},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("child_ids", err.Param)
}

func (suite *LocationSvcTestSuite) TestCreateLocation_RepoCreateError() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("db"), "insert failed")).
		Times(1)
	suite.expectLocCacheError()

	result, err := suite.locationSvc.CreateLocation(ctx, domain.CreateLocationParams{
		Name:     "Bin",
		TypeCode: "bin",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *LocationSvcTestSuite) TestCreateLocation_IdempotencyFinished() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))
	code := 200

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_loc_test",
			RecoveryPoint: string(domain.RecoveryPointFinished),
			ResponseCode:  &code,
			ResponseBody:  marshalJSON(&domain.Location{ID: "loc_cached", Name: "Bin Cached"}),
		}, nil).
		Times(1)

	result, err := suite.locationSvc.CreateLocation(ctx, domain.CreateLocationParams{Name: "Bin Cached"})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("loc_cached", result.ID)
}

func (suite *LocationSvcTestSuite) TestCreateLocation_UnexpectedRecoveryPoint() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_loc_test",
			RecoveryPoint: "unexpected_point",
		}, nil).
		Times(1)

	result, err := suite.locationSvc.CreateLocation(ctx, domain.CreateLocationParams{Name: "Bin"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

// ── UpdateLocation ────────────────────────────────────────────────────────────

func (suite *LocationSvcTestSuite) TestUpdateLocation_Success() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))
	newName := "Rack B"

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_u1").
		Return(true, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_u1"}).
		Return(&domain.Location{ID: "loc_u1", Name: "Rack A"}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.Location{ID: "loc_u1", Name: "Rack B"}, nil).
		Times(1)
	suite.expectLocCacheSuccess()

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{
		LocationID: "loc_u1",
		Name:       &newName,
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("Rack B", result.Name)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_MissingIdentity() {
	result, err := suite.locationSvc.UpdateLocation(context.Background(), domain.UpdateLocationParams{LocationID: "loc_u1"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_MissingTargetAccount() {
	ctx := idempotencyCtx(noTargetLocationCtx())

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{LocationID: "loc_u1"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_InsufficientPermissions() {
	ctx := readOnlyLocationCtx("ac_loc1")

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{LocationID: "loc_u1"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_LocationNotInAccount() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_foreign").
		Return(false, nil).
		Times(1)
	suite.expectLocCacheError()

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{
		LocationID: "loc_foreign",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_SelfParentRejected() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))
	parentID := "loc_u2"

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_u2").
		Return(true, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_u2"}).
		Return(&domain.Location{ID: "loc_u2", Name: "Self"}, nil).
		Times(1)
	suite.expectLocCacheError()

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{
		LocationID: "loc_u2",
		ParentID:   field.Set(parentID),
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("parent_id", err.Param)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_ParentNotInAccount() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))
	foreignParent := "loc_foreign_parent"

	suite.expectLocIdempotencyStarted()
	// First IsInAccount: verify the location being updated exists
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_u3").
		Return(true, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_u3"}).
		Return(&domain.Location{ID: "loc_u3", Name: "Bin"}, nil).
		Times(1)
	// Second IsInAccount: verify the new parent_id
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_foreign_parent").
		Return(false, nil).
		Times(1)
	suite.expectLocCacheError()

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{
		LocationID: "loc_u3",
		ParentID:   field.Set(foreignParent),
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("parent_id", err.Param)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_ChildNotInAccount() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_u4").
		Return(true, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_u4"}).
		Return(&domain.Location{ID: "loc_u4", Name: "Aisle"}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_foreign_child").
		Return(false, nil).
		Times(1)
	suite.expectLocCacheError()

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{
		LocationID: "loc_u4",
		ChildIDs:   field.Set([]string{"loc_foreign_child"}),
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("child_ids", err.Param)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_BackfillsExistingParentID() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))
	newName := "Updated"

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_u5").
		Return(true, nil).
		Times(1)
	// Location has existing parent
	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_u5"}).
		Return(&domain.Location{ID: "loc_u5", Name: "Old", ParentID: strPtr("loc_p")}, nil).
		Times(1)
	// Validate the backfilled parent exists (non-empty ParentID != LocationID path)
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_p").
		Return(true, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdateLocationParams) (*domain.Location, *apierror.APIError) {
			// ParentID must be backfilled from the existing location
			suite.Require().True(params.ParentID.IsSet())
			val, _ := params.ParentID.Value()
			suite.Equal("loc_p", val)
			return &domain.Location{ID: "loc_u5", Name: "Updated", ParentID: strPtr("loc_p")}, nil
		}).
		Times(1)
	suite.expectLocCacheSuccess()

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{
		LocationID: "loc_u5",
		Name:       &newName,
		// ParentID unset (not sent) → backfilled from existing location
	})

	suite.Nil(err)
	suite.NotNil(result)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_ClearsParent() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.expectLocIdempotencyStarted()
	suite.locationRepo.EXPECT().
		IsInAccount(gomock.Any(), "ac_loc1", "loc_u6").
		Return(true, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_u6"}).
		Return(&domain.Location{ID: "loc_u6", Name: "Bin", ParentID: strPtr("loc_p")}, nil).
		Times(1)
	// Explicit clear: IsInAccount should NOT be called for the parent.
	suite.locationRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdateLocationParams) (*domain.Location, *apierror.APIError) {
			suite.Require().True(params.ParentID.IsClear())
			return &domain.Location{ID: "loc_u6", Name: "Bin"}, nil
		}).
		Times(1)
	suite.expectLocCacheSuccess()

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{
		LocationID: "loc_u6",
		ParentID:   field.Clear[string](),
	})

	suite.Nil(err)
	suite.NotNil(result)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_IdempotencyFinished() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))
	code := 200

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_loc_test",
			RecoveryPoint: string(domain.RecoveryPointFinished),
			ResponseCode:  &code,
			ResponseBody:  marshalJSON(&domain.Location{ID: "loc_cached_u", Name: "Cached"}),
		}, nil).
		Times(1)

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{LocationID: "loc_u7"})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("loc_cached_u", result.ID)
}

func (suite *LocationSvcTestSuite) TestUpdateLocation_UnexpectedRecoveryPoint() {
	ctx := idempotencyCtx(internalLocationCtx("ac_loc1"))

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_loc_test",
			RecoveryPoint: "bad_point",
		}, nil).
		Times(1)

	result, err := suite.locationSvc.UpdateLocation(ctx, domain.UpdateLocationParams{LocationID: "loc_u8"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

// ── DeleteLocation ────────────────────────────────────────────────────────────

func (suite *LocationSvcTestSuite) TestDeleteLocation_Success() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_d1"}).
		Return(&domain.Location{ID: "loc_d1", Name: "Bin Del"}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		CountChildren(gomock.Any(), "ac_loc1", "loc_d1").
		Return(int64(0), nil).
		Times(1)
	suite.deletedRecordRepo.EXPECT().
		Create(gomock.Any(), constants.DeletedRecordResourceTypeLocation, "loc_d1", gomock.Any()).
		Return(nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Delete(gomock.Any(), domain.DeleteLocationParams{AccountID: "ac_loc1", LocationID: "loc_d1"}).
		Return(nil).
		Times(1)

	err := suite.locationSvc.DeleteLocation(ctx, domain.DeleteLocationParams{LocationID: "loc_d1"})

	suite.Nil(err)
}

func (suite *LocationSvcTestSuite) TestDeleteLocation_MissingIdentity() {
	err := suite.locationSvc.DeleteLocation(context.Background(), domain.DeleteLocationParams{LocationID: "loc_d2"})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *LocationSvcTestSuite) TestDeleteLocation_MissingTargetAccount() {
	err := suite.locationSvc.DeleteLocation(noTargetLocationCtx(), domain.DeleteLocationParams{LocationID: "loc_d2"})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (suite *LocationSvcTestSuite) TestDeleteLocation_InsufficientPermissions() {
	ctx := readOnlyLocationCtx("ac_loc1")

	err := suite.locationSvc.DeleteLocation(ctx, domain.DeleteLocationParams{LocationID: "loc_d2"})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *LocationSvcTestSuite) TestDeleteLocation_AlreadyDeleted() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_gone"}).
		Return(nil, apierror.NewResourceNotFoundError("not found")).
		Times(1)
	suite.deletedRecordRepo.EXPECT().
		Exists(gomock.Any(), constants.DeletedRecordResourceTypeLocation, "loc_gone").
		Return(true, nil).
		Times(1)

	err := suite.locationSvc.DeleteLocation(ctx, domain.DeleteLocationParams{LocationID: "loc_gone"})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceGone, err.Code)
}

func (suite *LocationSvcTestSuite) TestDeleteLocation_NotFound() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_nf2"}).
		Return(nil, apierror.NewResourceNotFoundError("not found")).
		Times(1)
	suite.deletedRecordRepo.EXPECT().
		Exists(gomock.Any(), constants.DeletedRecordResourceTypeLocation, "loc_nf2").
		Return(false, nil).
		Times(1)

	err := suite.locationSvc.DeleteLocation(ctx, domain.DeleteLocationParams{LocationID: "loc_nf2"})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

func (suite *LocationSvcTestSuite) TestDeleteLocation_HasChildren() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_parent"}).
		Return(&domain.Location{ID: "loc_parent", Name: "Aisle"}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		CountChildren(gomock.Any(), "ac_loc1", "loc_parent").
		Return(int64(3), nil).
		Times(1)

	err := suite.locationSvc.DeleteLocation(ctx, domain.DeleteLocationParams{LocationID: "loc_parent"})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "child locations")
}

func (suite *LocationSvcTestSuite) TestDeleteLocation_CountChildrenError() {
	ctx := internalLocationCtx("ac_loc1")

	suite.locationRepo.EXPECT().
		Get(gomock.Any(), domain.GetLocationParams{AccountID: "ac_loc1", LocationID: "loc_d_err"}).
		Return(&domain.Location{ID: "loc_d_err", Name: "Bin"}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		CountChildren(gomock.Any(), "ac_loc1", "loc_d_err").
		Return(int64(0), apierror.NewInternalError(errors.New("db"), "count query failed")).
		Times(1)

	err := suite.locationSvc.DeleteLocation(ctx, domain.DeleteLocationParams{LocationID: "loc_d_err"})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

// ── BulkUpsertLocations ───────────────────────────────────────────────────────

func (suite *LocationSvcTestSuite) TestBulkUpsertLocations_MissingIdentity() {
	result, err := suite.locationSvc.BulkUpsertLocations(context.Background(), domain.BulkUpsertLocationsParams{
		Locations: []domain.UpsertLocationParams{{Name: "Bin", TypeCode: "bin"}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *LocationSvcTestSuite) TestBulkUpsertLocations_MissingTargetAccount() {
	ctx := idempotencyCtx(noTargetLocationCtx())

	result, err := suite.locationSvc.BulkUpsertLocations(ctx, domain.BulkUpsertLocationsParams{
		Locations: []domain.UpsertLocationParams{{Name: "Bin", TypeCode: "bin"}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (suite *LocationSvcTestSuite) TestBulkUpsertLocations_InsufficientPermissions_NoCreate() {
	ctx := readOnlyLocationCtx("ac_loc1")

	result, err := suite.locationSvc.BulkUpsertLocations(ctx, domain.BulkUpsertLocationsParams{
		Locations: []domain.UpsertLocationParams{{Name: "Bin", TypeCode: "bin"}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *LocationSvcTestSuite) TestBulkUpsertLocations_InsufficientPermissions_NoUpdate() {
	ctx := createOnlyLocationCtx("ac_loc1")

	result, err := suite.locationSvc.BulkUpsertLocations(ctx, domain.BulkUpsertLocationsParams{
		Locations: []domain.UpsertLocationParams{{Name: "Bin", TypeCode: "bin"}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *LocationSvcTestSuite) TestBulkUpsertLocations_Empty() {
	ctx := internalLocationCtx("ac_loc1")

	result, err := suite.locationSvc.BulkUpsertLocations(ctx, domain.BulkUpsertLocationsParams{
		Locations: []domain.UpsertLocationParams{},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "No locations provided")
}

func (suite *LocationSvcTestSuite) TestBulkUpsertLocations_TooMany() {
	ctx := internalLocationCtx("ac_loc1")
	locs := make([]domain.UpsertLocationParams, 1001)

	result, err := suite.locationSvc.BulkUpsertLocations(ctx, domain.BulkUpsertLocationsParams{Locations: locs})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "1000")
}

func (suite *LocationSvcTestSuite) TestBulkUpsertLocations_DuplicateNameInRequest() {
	ctx := internalLocationCtx("ac_loc1")

	result, err := suite.locationSvc.BulkUpsertLocations(ctx, domain.BulkUpsertLocationsParams{
		Locations: []domain.UpsertLocationParams{
			{Name: "Bin A", TypeCode: "bin"},
			{Name: "bin a", TypeCode: "bin"}, // same name, different case
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("locations[1].name", err.Param)
	suite.Contains(err.PublicMessage, "duplicate name")
}

// --- resolveBulkUpsertLocationRows (the engine's Resolve hook, exercised directly) ---

// resolveLocations runs the Resolve hook against the mocked repo factory.
func (suite *LocationSvcTestSuite) resolveLocations(rows ...domain.UpsertLocationParams) ([]domain.ResolvedUpsertLocationRow, *apierror.APIError) {
	return resolveBulkUpsertLocationRows(internalLocationCtx("ac_loc1"), suite.repoFactory, "ac_loc1", rows)
}

// A parent reference that does not resolve fails fast with a row-indexed 400 in the accept
// phase, before any job is raised.
func (suite *LocationSvcTestSuite) TestResolveBulkUpsertLocations_UnknownParent() {
	suite.locationRepo.EXPECT().
		GetByIDs(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{}, nil). // the parent id resolves to nothing
		Times(1)

	resolved, err := suite.resolveLocations(domain.UpsertLocationParams{
		Name:     "Shelf",
		TypeCode: "shelf",
		Parent:   &domain.ObjectIdentifier{ID: "loc_missing"},
	})

	suite.Nil(resolved)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
}

// A reference to a sibling defined in the same batch is an intra-batch link: it resolves to
// a batch-name reference without touching the database, so an import can define its own
// hierarchy without the referenced rows pre-existing. No repo call is stubbed — reaching one
// would fail the test.
func (suite *LocationSvcTestSuite) TestResolveBulkUpsertLocations_IntraBatchReferences() {
	resolved, err := suite.resolveLocations(
		domain.UpsertLocationParams{Name: "Warehouse", TypeCode: "building", Children: []domain.ObjectIdentifier{{Name: "Aisle"}}},
		domain.UpsertLocationParams{Name: "Aisle", TypeCode: "aisle", Parent: &domain.ObjectIdentifier{Name: "warehouse"}}, // case-insensitive
	)

	suite.Nil(err)
	suite.Require().Len(resolved, 2)
	// Warehouse's child "Aisle" → batch-name reference.
	suite.Require().Len(resolved[0].Children, 1)
	suite.Equal("aisle", resolved[0].Children[0].BatchName)
	suite.Empty(resolved[0].Children[0].ExistingID)
	// Aisle's parent "warehouse" → batch-name reference (matched case-insensitively).
	suite.Require().NotNil(resolved[1].Parent)
	suite.Equal("warehouse", resolved[1].Parent.BatchName)
	suite.Empty(resolved[1].Parent.ExistingID)
}

// --- writeBulkUpsertLocations (the engine's Write hook, exercised directly) ---
//
// The accept phase and job plumbing are covered by the engine's own test; these rows prove
// the location-specific write logic: name matching, the create/update split, and the
// per-row parent/child linking. Write takes pre-resolved rows, so no resolver mock is set.

// writeLocations runs the Write hook against the mocked repo factory and returns the
// created/updated ids (from results) and per-row failures (from errors). apiErr is non-nil
// only for a pre-loop infrastructure failure (the bulk read); a row that fails its own
// upsert or link is recorded in rowErrs, not returned as apiErr.
func (suite *LocationSvcTestSuite) writeLocations(rows ...domain.ResolvedUpsertLocationRow) (created, updated []string, rowErrs []apierror.RowError, apiErr *apierror.APIError) {
	res, apiErr := writeBulkUpsertLocations(internalLocationCtx("ac_loc1"), suite.repoFactory, passthroughSavepoint{}, "ac_loc1", rows)
	if apiErr != nil {
		return nil, nil, nil, apiErr
	}
	created, updated = splitJobResults(res.Results)
	return created, updated, res.Errors, nil
}

func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_FindByNamesError() {
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("db"), "query failed")).
		Times(1)

	_, _, _, err := suite.writeLocations(domain.ResolvedUpsertLocationRow{Name: "Bin", TypeCode: "bin"})

	suite.NotNil(err, "a failed bulk read is infrastructural — it fails the whole batch")
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_AllCreates() {
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, locID string, _ domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
			return &domain.Location{ID: locID, Name: "Bin A"}, nil
		}).
		Times(1)

	created, updated, rowErrs, err := suite.writeLocations(domain.ResolvedUpsertLocationRow{Name: "Bin A", TypeCode: "bin"})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 1)
	suite.NotEmpty(created[0])
	suite.Nil(updated)
}

func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_AllUpdates() {
	existing := &domain.Location{ID: "loc_ex1", Name: "Bin A", TypeCode: "bin"}
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{existing}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.Location{ID: "loc_ex1", Name: "Bin A", TypeCode: "bin"}, nil).
		Times(1)

	created, updated, rowErrs, err := suite.writeLocations(domain.ResolvedUpsertLocationRow{Name: "Bin A", TypeCode: "bin"})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(updated, 1)
	suite.Equal("loc_ex1", updated[0])
	suite.Nil(created)
}

func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_MixedCreateAndUpdate() {
	existing := &domain.Location{ID: "loc_ex2", Name: "Bin B", TypeCode: "bin"}
	// "Bin B" exists; "Bin C" does not
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{existing}, nil).
		Times(1)
	// Create for "Bin C"
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, locID string, _ domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
			return &domain.Location{ID: locID, Name: "Bin C"}, nil
		}).
		Times(1)
	// Update for "Bin B"
	suite.locationRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.Location{ID: "loc_ex2", Name: "Bin B"}, nil).
		Times(1)

	created, updated, rowErrs, err := suite.writeLocations(
		domain.ResolvedUpsertLocationRow{Name: "Bin B", TypeCode: "bin"},
		domain.ResolvedUpsertLocationRow{Name: "Bin C", TypeCode: "bin"},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(updated, 1)
	suite.Equal("loc_ex2", updated[0])
	suite.Len(created, 1)
	suite.NotEmpty(created[0])
}

func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_ParentLinkApplied() {
	// "Shelf" is new; its resolved parent "loc_parent_p2" is a pre-existing location id.
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, locID string, _ domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
			return &domain.Location{ID: locID, Name: "Shelf"}, nil
		}).
		Times(1)
	// LinkParent is called with the created id as child and the resolved parent id.
	suite.locationRepo.EXPECT().
		LinkParent(gomock.Any(), "ac_loc1", gomock.Any(), "loc_parent_p2").
		Return(nil).
		Times(1)

	created, _, rowErrs, err := suite.writeLocations(domain.ResolvedUpsertLocationRow{
		Name:     "Shelf",
		TypeCode: "shelf",
		Parent:   &domain.LocationRef{ExistingID: "loc_parent_p2"},
	})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 1)
}

func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_ChildLinksApplied() {
	// "Aisle" is new; it specifies child_ids pointing to pre-existing locations.
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, locID string, _ domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
			return &domain.Location{ID: locID, Name: "Aisle"}, nil
		}).
		Times(1)
	// LinkParent is called for each child id (child → this location as parent).
	suite.locationRepo.EXPECT().
		LinkParent(gomock.Any(), "ac_loc1", "loc_child_a", gomock.Any()).
		Return(nil).
		Times(1)
	suite.locationRepo.EXPECT().
		LinkParent(gomock.Any(), "ac_loc1", "loc_child_b", gomock.Any()).
		Return(nil).
		Times(1)

	created, _, rowErrs, err := suite.writeLocations(domain.ResolvedUpsertLocationRow{
		Name:     "Aisle",
		TypeCode: "aisle",
		Children: []domain.LocationRef{{ExistingID: "loc_child_a"}, {ExistingID: "loc_child_b"}},
	})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 1)
}

// The location is upserted in phase 1; its parent link fails in phase 2. The link failure
// is recorded in errors while the location itself was still created — so the row appears in
// both results and errors. The batch does not fail.
func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_LinkFailureRecorded() {
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, locID string, _ domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
			return &domain.Location{ID: locID, Name: "Bin"}, nil
		}).
		Times(1)
	suite.locationRepo.EXPECT().
		LinkParent(gomock.Any(), "ac_loc1", gomock.Any(), "loc_parent_err").
		Return(apierror.NewInternalError(errors.New("db"), "link failed")).
		Times(1)

	created, updated, rowErrs, err := suite.writeLocations(domain.ResolvedUpsertLocationRow{
		Name:     "Bin",
		TypeCode: "bin",
		Parent:   &domain.LocationRef{ExistingID: "loc_parent_err"},
	})

	suite.Nil(err)
	suite.Len(created, 1, "the location was still upserted in phase 1")
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
	suite.Equal(0, rowErrIndex(rowErrs[0]))
}

func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_CreateFailureRecorded() {
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("db"), "insert failed")).
		Times(1)

	created, updated, rowErrs, err := suite.writeLocations(domain.ResolvedUpsertLocationRow{Name: "Bin Err", TypeCode: "bin"})

	suite.Nil(err)
	suite.Empty(created)
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
}

func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_UpdateFailureRecorded() {
	existing := &domain.Location{ID: "loc_ex_err", Name: "Bin Err", TypeCode: "bin"}
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{existing}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("db"), "update failed")).
		Times(1)

	created, updated, rowErrs, err := suite.writeLocations(domain.ResolvedUpsertLocationRow{Name: "Bin Err", TypeCode: "bin"})

	suite.Nil(err)
	suite.Empty(created)
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
}

func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_NoLinksWhenNoParentOrChildIDs() {
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, locID string, _ domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
			return &domain.Location{ID: locID, Name: "Root"}, nil
		}).
		Times(1)
	// LinkParent must NOT be called when no parent/child ids are provided.

	created, _, rowErrs, err := suite.writeLocations(domain.ResolvedUpsertLocationRow{Name: "Root", TypeCode: "warehouse"})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 1)
}

// Partial success: both rows upsert in phase 1; the bad row's parent link fails in phase 2
// and is recorded in errors. Both locations exist; only the bad row also carries an error.
func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_PartialSuccess() {
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, locID string, params domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
			return &domain.Location{ID: locID, Name: params.Name}, nil
		}).
		Times(2)
	// The bad row's parent link fails.
	suite.locationRepo.EXPECT().
		LinkParent(gomock.Any(), "ac_loc1", gomock.Any(), "loc_bad_parent").
		Return(apierror.NewInternalError(errors.New("db"), "link failed")).
		Times(1)

	created, updated, rowErrs, err := suite.writeLocations(
		domain.ResolvedUpsertLocationRow{Name: "Good", TypeCode: "bin"},
		domain.ResolvedUpsertLocationRow{Name: "Bad", TypeCode: "bin", Parent: &domain.LocationRef{ExistingID: "loc_bad_parent"}},
	)

	suite.Nil(err)
	suite.Len(created, 2, "both rows upsert in phase 1")
	suite.Empty(updated)
	suite.Len(rowErrs, 1, "only the bad row's link failed")
	suite.Equal(1, rowErrIndex(rowErrs[0]))
}

// Intra-batch link: "Aisle"'s parent names "Warehouse", a sibling created in the same
// batch. Phase 1 creates both; phase 2 links the aisle to the id phase 1 assigned the
// warehouse — proving a batch-name reference resolves against the batch, not the database.
func (suite *LocationSvcTestSuite) TestWriteBulkUpsertLocations_IntraBatchParentLink() {
	createdByName := make(map[string]string)
	suite.locationRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_loc1", gomock.Any()).
		Return([]*domain.Location{}, nil).
		Times(1)
	suite.locationRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, locID string, params domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
			createdByName[params.Name] = locID
			return &domain.Location{ID: locID, Name: params.Name}, nil
		}).
		Times(2)
	var linkChild, linkParent string
	suite.locationRepo.EXPECT().
		LinkParent(gomock.Any(), "ac_loc1", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, child, parent string) *apierror.APIError {
			linkChild, linkParent = child, parent
			return nil
		}).
		Times(1)

	created, _, rowErrs, err := suite.writeLocations(
		domain.ResolvedUpsertLocationRow{Name: "Warehouse", TypeCode: "building"},
		domain.ResolvedUpsertLocationRow{Name: "Aisle", TypeCode: "aisle", Parent: &domain.LocationRef{BatchName: "warehouse"}},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 2)
	suite.Equal(createdByName["Aisle"], linkChild, "the aisle is the child")
	suite.Equal(createdByName["Warehouse"], linkParent, "linked to the warehouse created in this batch")
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// reopens the export so assertions read what a spreadsheet would
func (suite *LocationSvcTestSuite) exportedRows(export *domain.Export) [][]string {
	return exportedSheetRows(suite.T(), export, "Storage Locations")
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *LocationSvcTestSuite) buildExport(ctx context.Context, params domain.ExportLocationsParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.locationSvc.(*locationSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}

func (suite *LocationSvcTestSuite) TestExportLocations_JoinsChildNames() {
	ctx := internalLocationCtx("ac_test123")
	warehouse := "Warehouse"
	suite.locationRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.Location{
		{
			ID:       "loc_1",
			Name:     "Warehouse",
			TypeCode: "warehouse",
			Children: []domain.LocationChild{
				{ID: "loc_2", Name: "Aisle 1"},
				{ID: "loc_3", Name: "Aisle 2"},
			},
		},
		{ID: "loc_2", Name: "Aisle 1", TypeCode: "aisle", ParentName: &warehouse},
	}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportLocationsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 3)
	suite.Equal([]string{"ID", "Name", "Type", "Parent", "Children"}, rows[0])
	// A root location has no parent but lists its children.
	suite.Equal([]string{"loc_1", "Warehouse", "warehouse", "", "Aisle 1; Aisle 2"}, rows[1])
	// A leaf names its parent and stops there once blanks are trimmed.
	suite.Equal([]string{"loc_2", "Aisle 1", "aisle", "Warehouse"}, rows[2])
}

func (suite *LocationSvcTestSuite) TestExportLocations_ScopesToTheIdentitysAccount() {
	ctx := internalLocationCtx("ac_owner")
	suite.locationRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportLocationsParams) ([]*domain.Location, error) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportLocationsParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}
