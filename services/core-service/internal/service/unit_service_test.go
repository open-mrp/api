package service

import (
	"context"
	"errors"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type stubOutboxRepo struct{}

func (s *stubOutboxRepo) Create(_ context.Context, _ messaging.OutboxMessageInput) (int64, error) {
	return 0, nil
}

type stubTxManager struct {
	factory domain.RepoFactory
}

func (m *stubTxManager) WithTx(ctx context.Context, fn func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	return fn(ctx, m.factory)
}

func (m *stubTxManager) WithTxSavepoint(ctx context.Context, fn func(context.Context, domain.RepoFactory, db.SavepointRunner) *apierror.APIError) *apierror.APIError {
	return fn(ctx, m.factory, passthroughSavepoint{})
}

// passthroughSavepoint runs each unit of work directly: the stubbed repos issue no real
// SQL, so there is nothing to bracket — a failing unit's error propagates and the engine
// records it, exactly as a real rolled-back savepoint would surface to the caller.
type passthroughSavepoint struct{}

func (passthroughSavepoint) Run(ctx context.Context, fn func(context.Context) *apierror.APIError) *apierror.APIError {
	return fn(ctx)
}

type UnitSvcTestSuite struct {
	suite.Suite
	unitSvc           domain.UnitSvc
	unitRepo          *repositorymock.MockUnitRepo
	deletedRecordRepo *repositorymock.MockDeletedRecordRepo
	repoFactory       *factorymock.MockRepoFactory
	mediatorFactory   *factorymock.MockMediatorFactory
	idempotencyMed    *mediatormock.MockIdempotencyMed
	ctrl              *gomock.Controller
}

func (suite *UnitSvcTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.unitRepo = repositorymock.NewMockUnitRepo(suite.ctrl)
	suite.deletedRecordRepo = repositorymock.NewMockDeletedRecordRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewUnitRepo().Return(suite.unitRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewDeletedRecordRepo().Return(suite.deletedRecordRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.unitSvc = NewUnitSvc(&UnitSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		// The async bulk upsert's accept phase fails fast on the rejection rows below
		// before it ever raises a job, and its write logic is tested directly via
		// writeBulkUpsertUnits; the engine's own test covers the job plumbing. So a real
		// factory that is never exercised satisfies the constructor.
		JobSvcFactory: NewJobSvcFactory(),
		TxManager:     &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *UnitSvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestUnitSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UnitSvcTestSuite))
}

func internalIdentityCtx(targetAccountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &targetAccountID,
			RoleType:     &adminCode,
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
	customCode := string(constants.RoleTypeCustom)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &targetAccountID,
			RoleType:     &customCode,
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
	adminCode := string(constants.RoleTypeAdmin)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			RoleType:     &adminCode,
			Permissions:  map[string]bool{"units:read": true},
		},
	})

	result, err := suite.unitSvc.GetUnit(ctx, "un_abc123")

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (suite *UnitSvcTestSuite) TestGetUnit_InsufficientPermissions() {
	customCode := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_test123"},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			RoleType:     &customCode,
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
			return &domain.Unit{ID: id, Name: "Gram", AccountID: new("ac_test123")}, nil
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
		Return(&domain.Unit{ID: "un_abc123", Name: "Old Name", AccountID: new("ac_test123")}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Updated Name", new("un_abc123")).
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
		Name:   new("Updated Name"),
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
		Name:   new("Test"),
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
		Name:   new("New Name"),
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
		Return(&domain.Unit{ID: "un_abc123", Name: "Old Name", AccountID: new("ac_test123")}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Kilogram", new("un_abc123")).
		Return(true, nil).
		Times(1)
	suite.expectCacheError()

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{
		UnitID: "un_abc123",
		Name:   new("Kilogram"),
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
		Return(&domain.Unit{ID: "un_abc123", Name: "Old Name", AccountID: new("ac_test123")}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		ExistsByAbbreviation(gomock.Any(), "ac_test123", "kg", new("un_abc123")).
		Return(true, nil).
		Times(1)
	suite.expectCacheError()

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{
		UnitID:       "un_abc123",
		Abbreviation: new("kg"),
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
		Return(&domain.Unit{ID: "un_abc123", Name: "Custom Unit", AccountID: new("ac_test123")}, nil).
		Times(1)
	suite.deletedRecordRepo.EXPECT().
		Create(gomock.Any(), constants.DeletedRecordResourceTypeUnit, "un_abc123", gomock.Any()).
		Return(nil).
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

	suite.deletedRecordRepo.EXPECT().
		Exists(gomock.Any(), constants.DeletedRecordResourceTypeUnit, "un_nonexistent").
		Return(false, nil).
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

func createOnlyIdentityCtx(targetAccountID string) context.Context {
	customCode := string(constants.RoleTypeCustom)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &targetAccountID,
			RoleType:     &customCode,
			Permissions: map[string]bool{
				"units:read":   true,
				"units:create": true,
			},
		},
	})
}

func (suite *UnitSvcTestSuite) TestDeleteUnit_ExternalActorRejected() {
	supplierCode := string(constants.RoleTypeAdmin)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_test123"},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeSupplier,
			ID:           "usr_external",
			RoleType:     &supplierCode,
			Permissions:  map[string]bool{"units:delete": true},
		},
	})

	err := suite.unitSvc.DeleteUnit(ctx, "un_abc123")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

// --- BulkUpsertUnits ---

func (suite *UnitSvcTestSuite) TestBulkUpsertUnits_MissingIdentity() {
	result, err := suite.unitSvc.BulkUpsertUnits(context.Background(), domain.BulkUpsertUnitsParams{
		Units: []domain.UpsertUnitParams{{Name: "Gram", Abbreviation: "g"}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitSvcTestSuite) TestBulkUpsertUnits_InsufficientPermissions_NoCreate() {
	ctx := readOnlyIdentityCtx("ac_test123")

	result, err := suite.unitSvc.BulkUpsertUnits(ctx, domain.BulkUpsertUnitsParams{
		Units: []domain.UpsertUnitParams{{Name: "Gram", Abbreviation: "g"}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitSvcTestSuite) TestBulkUpsertUnits_InsufficientPermissions_NoUpdate() {
	ctx := createOnlyIdentityCtx("ac_test123")

	result, err := suite.unitSvc.BulkUpsertUnits(ctx, domain.BulkUpsertUnitsParams{
		Units: []domain.UpsertUnitParams{{Name: "Gram", Abbreviation: "g"}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitSvcTestSuite) TestBulkUpsertUnits_EmptyUnits() {
	ctx := internalIdentityCtx("ac_test123")

	result, err := suite.unitSvc.BulkUpsertUnits(ctx, domain.BulkUpsertUnitsParams{
		Units: []domain.UpsertUnitParams{},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "No units provided")
}

func (suite *UnitSvcTestSuite) TestBulkUpsertUnits_TooManyUnits() {
	ctx := internalIdentityCtx("ac_test123")
	units := make([]domain.UpsertUnitParams, 1001)

	result, err := suite.unitSvc.BulkUpsertUnits(ctx, domain.BulkUpsertUnitsParams{Units: units})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "1000")
}

func (suite *UnitSvcTestSuite) TestBulkUpsertUnits_DuplicateNameInRequest() {
	ctx := internalIdentityCtx("ac_test123")

	result, err := suite.unitSvc.BulkUpsertUnits(ctx, domain.BulkUpsertUnitsParams{
		Units: []domain.UpsertUnitParams{
			{Name: "Gram", Abbreviation: "g"},
			{Name: "gram", Abbreviation: "grams"},
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("units[1].name", err.Param)
	suite.Contains(err.PublicMessage, "duplicate name")
}

func (suite *UnitSvcTestSuite) TestBulkUpsertUnits_DuplicateAbbreviationInRequest() {
	ctx := internalIdentityCtx("ac_test123")

	result, err := suite.unitSvc.BulkUpsertUnits(ctx, domain.BulkUpsertUnitsParams{
		Units: []domain.UpsertUnitParams{
			{Name: "Gram", Abbreviation: "g"},
			{Name: "GramUnit", Abbreviation: "G"},
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("units[1].abbreviation", err.Param)
	suite.Contains(err.PublicMessage, "duplicate abbreviation")
}

// --- writeBulkUpsertUnits (the engine's Write hook, exercised directly) ---
//
// The accept phase and job plumbing are covered by the engine's own test; these rows
// prove the units-specific write logic: dual-key matching, conflict detection,
// IsBaseUnit immutability, dimension immutability, and the created/updated split.

// writeUnits runs the Write hook against the suite's mocked repo factory and returns the
// created/updated ids (from results) and the per-row failures (from errors). apiErr is
// non-nil only for a pre-loop infrastructure failure (the bulk read, a data invariant);
// a row that fails its own upsert is recorded in rowErrs, not returned as apiErr.
func (suite *UnitSvcTestSuite) writeUnits(rows ...domain.UpsertUnitParams) (created, updated []string, rowErrs []apierror.RowError, apiErr *apierror.APIError) {
	// Identity in context: the write's audit publish attributes the event to the actor
	// that enqueued the work (the consumer restores it before executing the job).
	res, apiErr := writeBulkUpsertUnits(internalIdentityCtx("ac_test123"), suite.repoFactory, passthroughSavepoint{}, "ac_test123", rows)
	if apiErr != nil {
		return nil, nil, nil, apiErr
	}
	created, updated = splitJobResults(res.Results)
	return created, updated, res.Errors, nil
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_FindByAbbreviationsOrNamesError() {
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("db error"), "query failed")).
		Times(1)

	_, _, _, err := suite.writeUnits(domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g"})

	suite.NotNil(err, "a failed bulk read is infrastructural — it fails the whole batch")
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_EmptyIDInExistingUnit() {
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{{ID: "", Name: "Gram", Abbreviation: "g"}}, nil).
		Times(1)

	_, _, _, err := suite.writeUnits(domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g"})

	suite.NotNil(err, "a corrupt existing row is an invariant violation, not a per-row failure")
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_NameAndAbbrBelongToDifferentUnits() {
	accountID := "ac_test123"
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{
			{ID: "un_1", Name: "Gram", Abbreviation: "g", AccountID: &accountID},
			{ID: "un_2", Name: "Kilogram", Abbreviation: "kg", AccountID: &accountID},
		}, nil).
		Times(1)

	// "Gram" matches un_1 by name; "kg" matches un_2 by abbreviation — different units. The
	// conflicting row fails on its own and is recorded in errors; the batch does not error.
	created, updated, rowErrs, err := suite.writeUnits(domain.UpsertUnitParams{Name: "Gram", Abbreviation: "kg"})

	suite.Nil(err)
	suite.Empty(created)
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
	suite.Equal(0, rowErrIndex(rowErrs[0]))
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_CreateAlwaysSetsIsBaseUnitFalse() {
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, unitID string, params domain.CreateUnitParams) (*domain.Unit, *apierror.APIError) {
			suite.False(params.IsBaseUnit, "bulk upsert must never create a base unit")
			return &domain.Unit{ID: unitID, Name: "Gram", Abbreviation: "g"}, nil
		}).
		Times(1)

	created, _, rowErrs, err := suite.writeUnits(domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g", UnitDimensionCode: "mass"})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 1)
	suite.NotEmpty(created[0])
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_SystemUnitRejected() {
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{
			{ID: "un_sys", Name: "Gram", Abbreviation: "g", AccountID: nil},
		}, nil).
		Times(1)

	// Modifying a system unit is rejected per-row and recorded in errors.
	created, updated, rowErrs, err := suite.writeUnits(domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g"})

	suite.Nil(err)
	suite.Empty(created)
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
	suite.Contains(rowErrMessage(rowErrs[0]), "System units cannot be modified")
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_DimensionCodeImmutable() {
	accountID := "ac_test123"
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{
			{ID: "un_1", Name: "Gram", Abbreviation: "g", AccountID: &accountID, UnitDimensionCode: "mass"},
		}, nil).
		Times(1)

	created, updated, rowErrs, err := suite.writeUnits(domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g", UnitDimensionCode: "volume"})

	suite.Nil(err)
	suite.Empty(created)
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
	suite.Contains(rowErrMessage(rowErrs[0]), "immutable")
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_AllCreates() {
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, unitID string, _ domain.CreateUnitParams) (*domain.Unit, *apierror.APIError) {
			return &domain.Unit{ID: unitID, Name: "Gram", Abbreviation: "g"}, nil
		}).
		Times(1)

	created, updated, rowErrs, err := suite.writeUnits(domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g", UnitDimensionCode: "mass"})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 1)
	suite.NotEmpty(created[0])
	suite.Nil(updated)
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_AllUpdates() {
	accountID := "ac_test123"
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{
			{ID: "un_1", Name: "Gram", Abbreviation: "g", AccountID: &accountID, UnitDimensionCode: "mass"},
		}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.Unit{ID: "un_1", Name: "Gram", Abbreviation: "g"}, nil).
		Times(1)

	created, updated, rowErrs, err := suite.writeUnits(domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g", UnitDimensionCode: "mass"})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(updated, 1)
	suite.Equal("un_1", updated[0])
	suite.Nil(created)
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_MixedCreateAndUpdate() {
	accountID := "ac_test123"
	// "Kilogram"/"kg" exists; "Gram"/"g" is new
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{
			{ID: "un_kg", Name: "Kilogram", Abbreviation: "kg", AccountID: &accountID, UnitDimensionCode: "mass"},
		}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, unitID string, _ domain.CreateUnitParams) (*domain.Unit, *apierror.APIError) {
			return &domain.Unit{ID: unitID, Name: "Gram", Abbreviation: "g"}, nil
		}).
		Times(1)
	suite.unitRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.Unit{ID: "un_kg", Name: "Kilogram", Abbreviation: "kg"}, nil).
		Times(1)

	created, updated, rowErrs, err := suite.writeUnits(
		domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g", UnitDimensionCode: "mass"},
		domain.UpsertUnitParams{Name: "Kilogram", Abbreviation: "kg", UnitDimensionCode: "mass"},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 1)
	suite.NotEmpty(created[0])
	suite.Len(updated, 1)
	suite.Equal("un_kg", updated[0])
}

// Partial success: a good row is created while a bad row (immutable dimension change) is
// recorded in errors — the write does not fail the batch.
func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_PartialSuccess() {
	accountID := "ac_test123"
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{
			{ID: "un_kg", Name: "Kilogram", Abbreviation: "kg", AccountID: &accountID, UnitDimensionCode: "mass"},
		}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, unitID string, _ domain.CreateUnitParams) (*domain.Unit, *apierror.APIError) {
			return &domain.Unit{ID: unitID, Name: "Gram", Abbreviation: "g"}, nil
		}).
		Times(1)

	// Row 0 creates "Gram"; row 1 tries an immutable dimension change on "Kilogram".
	created, updated, rowErrs, err := suite.writeUnits(
		domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g", UnitDimensionCode: "mass"},
		domain.UpsertUnitParams{Name: "Kilogram", Abbreviation: "kg", UnitDimensionCode: "volume"},
	)

	suite.Nil(err)
	suite.Len(created, 1, "the valid row still writes")
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
	suite.Equal(1, rowErrIndex(rowErrs[0]))
	suite.Contains(rowErrMessage(rowErrs[0]), "immutable")
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_CreateRepoError() {
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("insert failed"), "db write error")).
		Times(1)

	// A row whose write fails is recorded in errors; the batch itself does not error.
	created, _, rowErrs, err := suite.writeUnits(domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g"})

	suite.Nil(err)
	suite.Empty(created)
	suite.Len(rowErrs, 1)
}

func (suite *UnitSvcTestSuite) TestWriteBulkUpsertUnits_UpdateRepoError() {
	accountID := "ac_test123"
	suite.unitRepo.EXPECT().
		FindByAbbreviationsOrNames(gomock.Any(), "ac_test123", gomock.Any(), gomock.Any()).
		Return([]*domain.Unit{
			{ID: "un_1", Name: "Gram", Abbreviation: "g", AccountID: &accountID, UnitDimensionCode: "mass"},
		}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("update failed"), "db write error")).
		Times(1)

	_, updated, rowErrs, err := suite.writeUnits(domain.UpsertUnitParams{Name: "Gram", Abbreviation: "g", UnitDimensionCode: "mass"})

	suite.Nil(err)
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
}

// --- isDenominatorZero ---

func (suite *UnitSvcTestSuite) TestIsDenominatorZero_ExactZero() {
	suite.True(isDenominatorZero("0"))
	suite.True(isDenominatorZero("0.0"))
	suite.True(isDenominatorZero("0.000000000000000000000000000000"))
	suite.True(isDenominatorZero("-0"))
}

func (suite *UnitSvcTestSuite) TestIsDenominatorZero_RoundsToZeroInDB() {
	// decimal(65,30) stores 30 fractional digits; anything with |x| < 5e-31
	// (half the smallest representable unit 1e-30) rounds to zero when stored.
	suite.True(isDenominatorZero("0.0000000000000000000000000000001"))  // 1e-31 < 5e-31
	suite.True(isDenominatorZero("0.00000000000000000000000000000049")) // 4.9e-31 < 5e-31
	suite.True(isDenominatorZero("-0.0000000000000000000000000000001")) // negative, same magnitude
}

func (suite *UnitSvcTestSuite) TestIsDenominatorZero_AtRoundingBoundary() {
	// 5e-31 is exactly half the smallest unit — MySQL rounds away from zero to 1e-30
	// (non-zero), so this must NOT be treated as zero.
	suite.False(isDenominatorZero("0.0000000000000000000000000000005")) // 5e-31, rounds up
}

func (suite *UnitSvcTestSuite) TestIsDenominatorZero_NonZero() {
	suite.False(isDenominatorZero("1"))
	suite.False(isDenominatorZero("1.0"))
	suite.False(isDenominatorZero("-1"))
	suite.False(isDenominatorZero("0.000000000000000000000000000001")) // 1e-30, smallest storable unit
}

func (suite *UnitSvcTestSuite) TestIsDenominatorZero_Unparseable() {
	suite.False(isDenominatorZero(""))
	suite.False(isDenominatorZero("abc"))
}

// --- CreateUnit denominator validation ---

func (suite *UnitSvcTestSuite) TestCreateUnit_ZeroRatioDenominator() {
	ctx := internalIdentityCtx("ac_test123")

	result, err := suite.unitSvc.CreateUnit(ctx, domain.CreateUnitParams{
		Name:              "Gram",
		Abbreviation:      "g",
		UnitDimensionCode: "mass",
		RatioNumerator:    "1",
		RatioDenominator:  "0",
		OffsetNumerator:   "0",
		OffsetDenominator: "1",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("ratio_denominator", err.Param)
}

func (suite *UnitSvcTestSuite) TestCreateUnit_ZeroOffsetDenominator() {
	ctx := internalIdentityCtx("ac_test123")

	result, err := suite.unitSvc.CreateUnit(ctx, domain.CreateUnitParams{
		Name:              "Gram",
		Abbreviation:      "g",
		UnitDimensionCode: "mass",
		RatioNumerator:    "1",
		RatioDenominator:  "1",
		OffsetNumerator:   "0",
		OffsetDenominator: "0",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("offset_denominator", err.Param)
}

// --- UpdateUnit denominator validation ---

func (suite *UnitSvcTestSuite) TestUpdateUnit_ZeroRatioDenominator() {
	ctx := internalIdentityCtx("ac_test123")

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{
		UnitID:           "un_abc123",
		RatioDenominator: new("0"),
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("ratio_denominator", err.Param)
}

func (suite *UnitSvcTestSuite) TestUpdateUnit_ZeroOffsetDenominator() {
	ctx := internalIdentityCtx("ac_test123")

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{
		UnitID:            "un_abc123",
		OffsetDenominator: new("0"),
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("offset_denominator", err.Param)
}

func (suite *UnitSvcTestSuite) TestUpdateUnit_NilDenominatorsSkipCheck() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_test123"))

	suite.expectIdempotencyStarted()
	suite.unitRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitParams{AccountID: "ac_test123", UnitID: "un_abc123"}).
		Return(&domain.Unit{ID: "un_abc123", Name: "Old Name", AccountID: new("ac_test123")}, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "New Name", new("un_abc123")).
		Return(false, nil).
		Times(1)
	suite.unitRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.Unit{ID: "un_abc123", Name: "New Name"}, nil).
		Times(1)
	suite.expectCacheSuccess()

	result, err := suite.unitSvc.UpdateUnit(ctx, domain.UpdateUnitParams{
		UnitID:            "un_abc123",
		Name:              new("New Name"),
		RatioDenominator:  nil,
		OffsetDenominator: nil,
	})

	suite.Nil(err)
	suite.NotNil(result)
}

// --- BulkUpsertUnits denominator validation ---

func (suite *UnitSvcTestSuite) TestBulkUpsertUnits_ZeroRatioDenominator() {
	ctx := internalIdentityCtx("ac_test123")

	result, err := suite.unitSvc.BulkUpsertUnits(ctx, domain.BulkUpsertUnitsParams{
		Units: []domain.UpsertUnitParams{
			{
				Name:              "Gram",
				Abbreviation:      "g",
				UnitDimensionCode: "mass",
				RatioNumerator:    "1",
				RatioDenominator:  "0",
				OffsetNumerator:   "0",
				OffsetDenominator: "1",
			},
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("units[0].ratio_denominator", err.Param)
	suite.Contains(err.PublicMessage, "ratio denominator cannot be zero")
}

func (suite *UnitSvcTestSuite) TestBulkUpsertUnits_ZeroOffsetDenominator() {
	ctx := internalIdentityCtx("ac_test123")

	result, err := suite.unitSvc.BulkUpsertUnits(ctx, domain.BulkUpsertUnitsParams{
		Units: []domain.UpsertUnitParams{
			{
				Name:              "Gram",
				Abbreviation:      "g",
				UnitDimensionCode: "mass",
				RatioNumerator:    "1",
				RatioDenominator:  "1",
				OffsetNumerator:   "0",
				OffsetDenominator: "0",
			},
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("units[0].offset_denominator", err.Param)
	suite.Contains(err.PublicMessage, "offset denominator cannot be zero")
}

func (suite *UnitSvcTestSuite) TestBulkUpsertUnits_ZeroDenominatorOnSecondUnit() {
	ctx := internalIdentityCtx("ac_test123")

	result, err := suite.unitSvc.BulkUpsertUnits(ctx, domain.BulkUpsertUnitsParams{
		Units: []domain.UpsertUnitParams{
			{Name: "Gram", Abbreviation: "g", RatioDenominator: "1", OffsetDenominator: "1"},
			{Name: "Kilogram", Abbreviation: "kg", RatioDenominator: "0", OffsetDenominator: "1"},
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("units[1].ratio_denominator", err.Param)
	suite.Contains(err.PublicMessage, "ratio denominator cannot be zero")
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

func (suite *UnitSvcTestSuite) exportedRows(export *domain.Export) [][]string {
	return exportedSheetRows(suite.T(), export, "Units")
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *UnitSvcTestSuite) buildExport(ctx context.Context, params domain.ExportUnitsParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.unitSvc.(*unitSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}

func (suite *UnitSvcTestSuite) TestExportUnits_WritesHeadersAndRows() {
	ctx := internalIdentityCtx("ac_test123")
	accountID := "ac_test123"
	suite.unitRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.Unit{
		// Padded exactly as MySQL hands a DECIMAL back, so the trimming is exercised.
		{
			ID: "unt_1", Name: "Kilogram", Abbreviation: "kg", UnitDimensionCode: "mass",
			RatioNumerator:    "1.000000000000000000000000000000",
			RatioDenominator:  "1.000000000000000000000000000000",
			OffsetNumerator:   "0.000000000000000000000000000000",
			OffsetDenominator: "1.000000000000000000000000000000",
			AccountID:         &accountID,
		},
		{
			ID: "unt_2", Name: "Gram", Abbreviation: "g", UnitDimensionCode: "mass",
			RatioNumerator:    "1.000000000000000000000000000000",
			RatioDenominator:  "1000.500000000000000000000000000000",
			OffsetNumerator:   "0.000000000000000000000000000000",
			OffsetDenominator: "1.000000000000000000000000000000",
		},
	}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportUnitsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 3)
	suite.Equal([]string{
		"ID", "Name", "Abbreviation", "Type", "Ratio Numerator", "Ratio Denominator",
		"Offset Numerator", "Offset Denominator", "Default",
	}, rows[0])
	// The stored scale never reaches the sheet; a real fraction keeps its digits.
	suite.Equal([]string{"unt_1", "Kilogram", "kg", "mass", "1", "1", "0", "1", "Yes"}, rows[1])
	// A system unit has no owning account, which is what the "Default" column reports.
	suite.Equal([]string{"unt_2", "Gram", "g", "mass", "1", "1000.5", "0", "1", "No"}, rows[2])
}

func (suite *UnitSvcTestSuite) TestExportUnits_ScopesToTheIdentitysAccount() {
	ctx := internalIdentityCtx("ac_owner")
	suite.unitRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportUnitsParams) ([]*domain.Unit, error) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportUnitsParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}
