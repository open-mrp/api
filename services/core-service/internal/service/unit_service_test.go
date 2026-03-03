package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type stubTxManager struct {
	factory domain.RepoFactory
}

func (m *stubTxManager) WithTx(_ context.Context, fn func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	return fn(context.Background(), m.factory)
}

type UnitSvcTestSuite struct {
	suite.Suite
	unitSvc         domain.UnitSvc
	unitRepo        *repositorymock.MockUnitRepo
	repoFactory     *factorymock.MockRepoFactory
	mediatorFactory *factorymock.MockMediatorFactory
	idempotencyMed  *mediatormock.MockIdempotencyMed
	ctrl            *gomock.Controller
}

func (suite *UnitSvcTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.unitRepo = repositorymock.NewMockUnitRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewUnitRepo().Return(suite.unitRepo).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.unitSvc = NewUnitSvc(&UnitSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *UnitSvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestUnitSvcTestSuite(t *testing.T) {
	suite.Run(t, new(UnitSvcTestSuite))
}

func strPtr(s string) *string { return &s }

func internalIdentityCtx(targetAccountID string) context.Context {
	adminCode := string(constants.RoleTypeCodeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: &targetAccountID,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           "usr_test123",
			RoleTypeCode: &adminCode,
			Permissions: map[string]bool{
				"units:read":   true,
				"units:create": true,
				"units:update": true,
				"units:delete": true,
			},
		},
	})
}

func idempotencyCtx(ctx context.Context) context.Context {
	ctx = appctx.WithIdempotencyKey(ctx, "test-idempotency-key")
	ctx = appctx.WithHandler(ctx, "/core.CoreService/TestHandler")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx, &appctx.IdempotencyResponseMetadata{})
	return ctx
}

func readOnlyIdentityCtx(targetAccountID string) context.Context {
	customCode := string(constants.RoleTypeCodeCustom)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: &targetAccountID,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           "usr_test123",
			RoleTypeCode: &customCode,
			Permissions: map[string]bool{
				"units:read": true,
			},
		},
	})
}

func (suite *UnitSvcTestSuite) expectIdempotencyStarted() {
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_test",
			RecoveryPoint: string(domain.RecoveryPointStarted),
		}, nil).
		Times(1)
}

func (suite *UnitSvcTestSuite) expectCacheSuccess() {
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), "idk_test", gomock.Any()).
		Return(nil).
		Times(1)
}

func (suite *UnitSvcTestSuite) expectCacheError() {
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), "idk_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		}).
		Times(1)
}

// --- GetUnit ---

func (suite *UnitSvcTestSuite) TestGetUnit_Success() {
	ctx := internalIdentityCtx("ac_test123")

	expected := &domain.Unit{ID: "un_abc123", Name: "Kilogram"}
	suite.unitRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitParams{AccountID: "ac_test123", UnitID: "un_abc123"}).
		Return(expected, nil).
		Times(1)

	result, err := suite.unitSvc.GetUnit(ctx, "un_abc123")

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("un_abc123", result.ID)
}

func (suite *UnitSvcTestSuite) TestGetUnit_MissingIdentity() {
	result, err := suite.unitSvc.GetUnit(context.Background(), "un_abc123")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitSvcTestSuite) TestGetUnit_MissingTargetAccount() {
	adminCode := string(constants.RoleTypeCodeAdmin)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityTypeUser,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           "usr_test123",
			RoleTypeCode: &adminCode,
			Permissions:  map[string]bool{"units:read": true},
		},
	})

	result, err := suite.unitSvc.GetUnit(ctx, "un_abc123")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (suite *UnitSvcTestSuite) TestGetUnit_InsufficientPermissions() {
	customCode := string(constants.RoleTypeCodeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: strPtr("ac_test123"),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           "usr_test123",
			RoleTypeCode: &customCode,
			Permissions:  map[string]bool{},
		},
	})

	result, err := suite.unitSvc.GetUnit(ctx, "un_abc123")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitSvcTestSuite) TestGetUnit_RepoError() {
	ctx := internalIdentityCtx("ac_test123")

	suite.unitRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewResourceNotFoundError("Unit not found.")).
		Times(1)

	result, err := suite.unitSvc.GetUnit(ctx, "un_nonexistent")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

// --- CreateUnit ---

func (suite *UnitSvcTestSuite) TestCreateUnit_Success() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_test123"))

	suite.expectIdempotencyStarted()
	suite.unitRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Gram", (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		ExistsByAbbreviation(gomock.Any(), "ac_test123", "g", (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, params domain.CreateUnitParams) (*domain.Unit, *apierror.APIError) {
			suite.Equal("ac_test123", params.AccountID)
			suite.Equal("Gram", params.Name)
			return &domain.Unit{ID: id, Name: "Gram", AccountID: strPtr("ac_test123")}, nil
		}).
		Times(1)
	suite.expectCacheSuccess()

	result, err := suite.unitSvc.CreateUnit(ctx, domain.CreateUnitParams{
		Name:              "Gram",
		Abbreviation:      "g",
		UnitDimensionCode: "mass",
		RatioNumerator:    "1",
		RatioDenominator:  "1",
		OffsetNumerator:   "0",
		OffsetDenominator: "1",
		IsBaseUnit:        true,
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("Gram", result.Name)
	suite.NotEmpty(result.ID)
}

func (suite *UnitSvcTestSuite) TestCreateUnit_MissingIdentity() {
	result, err := suite.unitSvc.CreateUnit(context.Background(), domain.CreateUnitParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitSvcTestSuite) TestCreateUnit_InsufficientPermissions() {
	ctx := readOnlyIdentityCtx("ac_test123")

	result, err := suite.unitSvc.CreateUnit(ctx, domain.CreateUnitParams{Name: "Test"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitSvcTestSuite) TestCreateUnit_RepoError() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_test123"))

	suite.expectIdempotencyStarted()
	suite.unitRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Gram", (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		ExistsByAbbreviation(gomock.Any(), "ac_test123", gomock.Any(), (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewConflictErrorWithParam("A unit with this name already exists.", "name")).
		Times(1)
	suite.expectCacheError()

	result, err := suite.unitSvc.CreateUnit(ctx, domain.CreateUnitParams{Name: "Gram"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceConflict, err.Code)
	suite.Equal("name", err.Param)
}

func (suite *UnitSvcTestSuite) TestCreateUnit_DuplicateName() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_test123"))

	suite.expectIdempotencyStarted()
	suite.unitRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Kilogram", (*string)(nil)).
		Return(true, nil).
		Times(1)
	suite.expectCacheError()

	result, err := suite.unitSvc.CreateUnit(ctx, domain.CreateUnitParams{
		Name:         "Kilogram",
		Abbreviation: "kg",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceConflict, err.Code)
	suite.Equal("name", err.Param)
}

func (suite *UnitSvcTestSuite) TestCreateUnit_DuplicateAbbreviation() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_test123"))

	suite.expectIdempotencyStarted()
	suite.unitRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "My Unit", (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		ExistsByAbbreviation(gomock.Any(), "ac_test123", "kg", (*string)(nil)).
		Return(true, nil).
		Times(1)
	suite.expectCacheError()

	result, err := suite.unitSvc.CreateUnit(ctx, domain.CreateUnitParams{
		Name:         "My Unit",
		Abbreviation: "kg",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceConflict, err.Code)
	suite.Equal("abbreviation", err.Param)
}

// --- UpdateUnit ---

func (suite *UnitSvcTestSuite) TestUpdateUnit_Success() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_test123"))

	suite.expectIdempotencyStarted()
	suite.unitRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitParams{AccountID: "ac_test123", UnitID: "un_abc123"}).
		Return(&domain.Unit{ID: "un_abc123", Name: "Old Name", AccountID: strPtr("ac_test123")}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Updated Name", strPtr("un_abc123")).
		Return(false, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdateUnitParams) (*domain.Unit, *apierror.APIError) {
			suite.Equal("ac_test123", params.AccountID)
			suite.Equal("un_abc123", params.UnitID)
			suite.Equal("Updated Name", *params.Name)
			return &domain.Unit{ID: "un_abc123", Name: "Updated Name"}, nil
		}).
		Times(1)
	suite.expectCacheSuccess()

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{
		UnitID: "un_abc123",
		Name:   strPtr("Updated Name"),
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("Updated Name", result.Name)
}

func (suite *UnitSvcTestSuite) TestUpdateUnit_MissingIdentity() {
	result, err := suite.unitSvc.UpdateUnit(context.Background(), domain.UpdateUnitParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitSvcTestSuite) TestUpdateUnit_InsufficientPermissions() {
	ctx := readOnlyIdentityCtx("ac_test123")

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{UnitID: "un_abc123"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitSvcTestSuite) TestUpdateUnit_NotFound() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_test123"))

	suite.expectIdempotencyStarted()
	suite.unitRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitParams{AccountID: "ac_test123", UnitID: "un_nonexistent"}).
		Return(nil, apierror.NewResourceNotFoundError("Unit not found.")).
		Times(1)
	suite.expectCacheError()

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{
		UnitID: "un_nonexistent",
		Name:   strPtr("Test"),
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

func (suite *UnitSvcTestSuite) TestUpdateUnit_SystemUnitRejected() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_test123"))

	suite.expectIdempotencyStarted()
	suite.unitRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitParams{AccountID: "ac_test123", UnitID: "un_system"}).
		Return(&domain.Unit{ID: "un_system", Name: "Kilogram", AccountID: nil}, nil).
		Times(1)
	suite.expectCacheError()

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{
		UnitID: "un_system",
		Name:   strPtr("New Name"),
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "System units cannot be modified")
}

func (suite *UnitSvcTestSuite) TestUpdateUnit_DuplicateName() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_test123"))

	suite.expectIdempotencyStarted()
	suite.unitRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitParams{AccountID: "ac_test123", UnitID: "un_abc123"}).
		Return(&domain.Unit{ID: "un_abc123", Name: "Old Name", AccountID: strPtr("ac_test123")}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Kilogram", strPtr("un_abc123")).
		Return(true, nil).
		Times(1)
	suite.expectCacheError()

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{
		UnitID: "un_abc123",
		Name:   strPtr("Kilogram"),
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceConflict, err.Code)
	suite.Equal("name", err.Param)
}

func (suite *UnitSvcTestSuite) TestUpdateUnit_DuplicateAbbreviation() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_test123"))

	suite.expectIdempotencyStarted()
	suite.unitRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitParams{AccountID: "ac_test123", UnitID: "un_abc123"}).
		Return(&domain.Unit{ID: "un_abc123", Name: "Old Name", AccountID: strPtr("ac_test123")}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		ExistsByAbbreviation(gomock.Any(), "ac_test123", "kg", strPtr("un_abc123")).
		Return(true, nil).
		Times(1)
	suite.expectCacheError()

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{
		UnitID:       "un_abc123",
		Abbreviation: strPtr("kg"),
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceConflict, err.Code)
	suite.Equal("abbreviation", err.Param)
}

// --- DeleteUnit ---

func (suite *UnitSvcTestSuite) TestDeleteUnit_Success() {
	ctx := internalIdentityCtx("ac_test123")

	suite.unitRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitParams{AccountID: "ac_test123", UnitID: "un_abc123"}).
		Return(&domain.Unit{ID: "un_abc123", Name: "Custom Unit", AccountID: strPtr("ac_test123")}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Delete(gomock.Any(), domain.DeleteUnitParams{AccountID: "ac_test123", UnitID: "un_abc123"}).
		Return(nil).
		Times(1)

	err := suite.unitSvc.DeleteUnit(ctx, "un_abc123")

	suite.Nil(err)
}

func (suite *UnitSvcTestSuite) TestDeleteUnit_MissingIdentity() {
	err := suite.unitSvc.DeleteUnit(context.Background(), "un_abc123")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitSvcTestSuite) TestDeleteUnit_InsufficientPermissions() {
	ctx := readOnlyIdentityCtx("ac_test123")

	err := suite.unitSvc.DeleteUnit(ctx, "un_abc123")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitSvcTestSuite) TestDeleteUnit_NotFound() {
	ctx := internalIdentityCtx("ac_test123")

	suite.unitRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitParams{AccountID: "ac_test123", UnitID: "un_nonexistent"}).
		Return(nil, apierror.NewResourceNotFoundError("Unit not found.")).
		Times(1)

	err := suite.unitSvc.DeleteUnit(ctx, "un_nonexistent")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

func (suite *UnitSvcTestSuite) TestDeleteUnit_SystemUnitRejected() {
	ctx := internalIdentityCtx("ac_test123")

	suite.unitRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitParams{AccountID: "ac_test123", UnitID: "un_system"}).
		Return(&domain.Unit{ID: "un_system", Name: "Kilogram", AccountID: nil}, nil).
		Times(1)

	err := suite.unitSvc.DeleteUnit(ctx, "un_system")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "System units cannot be deleted")
}

func (suite *UnitSvcTestSuite) TestDeleteUnit_ExternalActorRejected() {
	supplierCode := string(constants.RoleTypeCodeAdmin)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: strPtr("ac_test123"),
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeSupplier,
			ID:           "usr_external",
			RoleTypeCode: &supplierCode,
			Permissions:  map[string]bool{"units:delete": true},
		},
	})

	err := suite.unitSvc.DeleteUnit(ctx, "un_abc123")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}
