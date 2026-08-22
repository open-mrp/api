package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// --- UnitGroupSvcTestSuite ---

type UnitGroupSvcTestSuite struct {
	suite.Suite
	unitGroupSvc      domain.UnitGroupSvc
	unitGroupRepo     *repositorymock.MockUnitGroupRepo
	unitRepo          *repositorymock.MockUnitRepo
	unitQueryRepo     *repositorymock.MockUnitQueryRepo
	deletedRecordRepo *repositorymock.MockDeletedRecordRepo
	repoFactory       *factorymock.MockRepoFactory
	mediatorFactory   *factorymock.MockMediatorFactory
	idempotencyMed    *mediatormock.MockIdempotencyMed
	ctrl              *gomock.Controller
}

// SetupTest builds fresh mocks and a fresh controller for every test so that
// expectations from one test never leak into another.
func (suite *UnitGroupSvcTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.unitGroupRepo = repositorymock.NewMockUnitGroupRepo(suite.ctrl)
	suite.unitRepo = repositorymock.NewMockUnitRepo(suite.ctrl)
	suite.unitQueryRepo = repositorymock.NewMockUnitQueryRepo(suite.ctrl)
	suite.deletedRecordRepo = repositorymock.NewMockDeletedRecordRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewUnitGroupRepo().Return(suite.unitGroupRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewUnitRepo().Return(suite.unitRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewUnitQueryRepo().Return(suite.unitQueryRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewDeletedRecordRepo().Return(suite.deletedRecordRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.unitGroupSvc = NewUnitGroupSvc(&UnitGroupSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
		// The accept-rejection tests fail before the job is raised and the write tests call
		// writeBulkUpsertUnitGroups directly, so the factory is never exercised here (the
		// engine's own test covers the job plumbing); a real one satisfies validate().
		JobSvcFactory: NewJobSvcFactory(),
	})
}

func (suite *UnitGroupSvcTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestUnitGroupSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UnitGroupSvcTestSuite))
}

// --- identity context helpers ---

func internalUnitGroupCtx(targetAccountID string) context.Context {
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
				"unit_groups:read":   true,
				"unit_groups:create": true,
				"unit_groups:update": true,
				"unit_groups:delete": true,
			},
		},
	})
}

func readOnlyUnitGroupCtx(targetAccountID string) context.Context {
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
				"unit_groups:read": true,
			},
		},
	})
}

func createOnlyUnitGroupCtx(targetAccountID string) context.Context {
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
				"unit_groups:read":   true,
				"unit_groups:create": true,
			},
		},
	})
}

// --- idempotency helpers ---

func (suite *UnitGroupSvcTestSuite) expectUGIdempotencyStarted() {
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_ug_test",
			RecoveryPoint: string(domain.RecoveryPointStarted),
		}, nil).
		Times(1)
}

func (suite *UnitGroupSvcTestSuite) expectUGCacheSuccess() {
	suite.idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), "idk_ug_test", gomock.Any()).
		Return(nil).
		Times(1)
}

func (suite *UnitGroupSvcTestSuite) expectUGCacheError() {
	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), "idk_ug_test", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		}).
		Times(1)
}

// expectUnitsResolveByID stubs the unit-identifier resolver's id lookup for a bulk upsert:
// every requested unit id comes back with dimension dim, except those named in
// dimByID (a unit id mapped to "" is treated as missing, so its identifier fails to resolve).
func (suite *UnitGroupSvcTestSuite) expectUnitsResolveByID(dim string, dimByID map[string]string) {
	suite.unitRepo.EXPECT().
		GetByIDs(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, ids []string) ([]*domain.Unit, *apierror.APIError) {
			units := make([]*domain.Unit, 0, len(ids))
			for _, unitID := range ids {
				unitDim, ok := dimByID[unitID]
				if !ok {
					unitDim = dim
				}
				if unitDim == "" {
					continue // absent from the result set → the identifier does not resolve
				}
				units = append(units, &domain.Unit{ID: unitID, UnitDimensionCode: unitDim})
			}
			return units, nil
		}).
		Times(1)
}

// --- helper types ---

func ugAccountID(id string) *string { return &id }

// --- GetUnitGroup ---

func (suite *UnitGroupSvcTestSuite) TestGetUnitGroup_Success() {
	ctx := internalUnitGroupCtx("ac_test123")
	expected := &domain.UnitGroupFull{ID: "ug_abc123", Name: "Mass Group", AccountID: ugAccountID("ac_test123")}

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_abc123"}).
		Return(expected, nil).
		Times(1)

	result, err := suite.unitGroupSvc.GetUnitGroup(ctx, domain.GetUnitGroupParams{UnitGroupID: "ug_abc123"})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("ug_abc123", result.ID)
}

func (suite *UnitGroupSvcTestSuite) TestGetUnitGroup_MissingIdentity() {
	result, err := suite.unitGroupSvc.GetUnitGroup(context.Background(), domain.GetUnitGroupParams{UnitGroupID: "ug_abc123"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestGetUnitGroup_MissingTargetAccount() {
	customCode := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			RoleType:     &customCode,
			Permissions:  map[string]bool{"unit_groups:read": true},
		},
	})

	result, err := suite.unitGroupSvc.GetUnitGroup(ctx, domain.GetUnitGroupParams{UnitGroupID: "ug_abc123"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestGetUnitGroup_InsufficientPermissions() {
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

	result, err := suite.unitGroupSvc.GetUnitGroup(ctx, domain.GetUnitGroupParams{UnitGroupID: "ug_abc123"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestGetUnitGroup_NotFound() {
	ctx := internalUnitGroupCtx("ac_test123")

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_notfound"}).
		Return(nil, apierror.NewResourceNotFoundError("Unit group not found.")).
		Times(1)

	result, err := suite.unitGroupSvc.GetUnitGroup(ctx, domain.GetUnitGroupParams{UnitGroupID: "ug_notfound"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

// --- ListUnitGroups ---

func (suite *UnitGroupSvcTestSuite) TestListUnitGroups_Success_Empty() {
	ctx := internalUnitGroupCtx("ac_test123")

	suite.unitGroupRepo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(&domain.ListUnitGroupsResult{UnitGroups: []*domain.UnitGroupFull{}}, nil).
		Times(1)

	result, err := suite.unitGroupSvc.ListUnitGroups(ctx, domain.ListUnitGroupsParams{})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Empty(result.UnitGroups)
}

func (suite *UnitGroupSvcTestSuite) TestListUnitGroups_MissingIdentity() {
	result, err := suite.unitGroupSvc.ListUnitGroups(context.Background(), domain.ListUnitGroupsParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestListUnitGroups_InsufficientPermissions() {
	customCode := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_test123"},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			RoleType:     &customCode,
			Permissions:  map[string]bool{}, // no unit_groups:read
		},
	})

	result, err := suite.unitGroupSvc.ListUnitGroups(ctx, domain.ListUnitGroupsParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestListUnitGroups_Success_WithGroups() {
	ctx := internalUnitGroupCtx("ac_test123")
	accountID := "ac_test123"
	group := &domain.UnitGroupFull{ID: "ug_abc123", Name: "Mass Group", AccountID: &accountID}

	suite.unitGroupRepo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(&domain.ListUnitGroupsResult{UnitGroups: []*domain.UnitGroupFull{group}}, nil).
		Times(1)

	result, err := suite.unitGroupSvc.ListUnitGroups(ctx, domain.ListUnitGroupsParams{})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Len(result.UnitGroups, 1)
	suite.Equal("ug_abc123", result.UnitGroups[0].ID)
}

// --- CreateUnitGroup ---

func (suite *UnitGroupSvcTestSuite) TestCreateUnitGroup_Success_NoConversions() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))
	accountID := "ac_test123"

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Mass Group", (*string)(nil)).
		Return(false, nil).
		Times(1)
	// The base unit's dimension is validated against the group's type.
	suite.unitQueryRepo.EXPECT().
		Find(gomock.Any(), "ac_test123", "un_base").
		Return(&domain.LightUnit{ID: "un_base", Type: "mass"}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_new", Name: "Mass Group", AccountID: &accountID}, nil).
		Times(1)
	// The base unit is auto-included as an associated unit.
	suite.unitGroupRepo.EXPECT().
		UpsertUnitGroupUnit(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupUnit{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_new", Name: "Mass Group", AccountID: &accountID}, nil).
		Times(1)
	suite.expectUGCacheSuccess()

	result, err := suite.unitGroupSvc.CreateUnitGroup(ctx, domain.CreateUnitGroupParams{
		Name:       "Mass Group",
		Type:       "mass",
		BaseUnitID: "un_base",
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("ug_new", result.ID)
}

func (suite *UnitGroupSvcTestSuite) TestCreateUnitGroup_Success_WithConversions() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))
	accountID := "ac_test123"

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Mass Group2", (*string)(nil)).
		Return(false, nil).
		Times(1)
	// Find is called once per conversion (validateUnitConversionTypes) and once for the
	// base unit's dimension check; here both the conversion and the base unit are "un_kg".
	suite.unitQueryRepo.EXPECT().
		Find(gomock.Any(), "ac_test123", "un_kg").
		Return(&domain.LightUnit{ID: "un_kg", Type: "mass"}, nil).
		Times(2)
	suite.unitGroupRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_new2", Name: "Mass Group2", AccountID: &accountID}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		UpsertUnitGroupUnit(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupUnit{ID: "ugu_2", UnitID: "un_kg", UnitGroupID: "ug_new2"}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{
			ID:        "ug_new2",
			Name:      "Mass Group2",
			AccountID: &accountID,
			UnitConversions: []*domain.UnitGroupUnit{
				{ID: "ugu_2", UnitID: "un_kg"},
			},
		}, nil).
		Times(1)
	suite.expectUGCacheSuccess()

	result, err := suite.unitGroupSvc.CreateUnitGroup(ctx, domain.CreateUnitGroupParams{
		Name:       "Mass Group2",
		Type:       "mass",
		BaseUnitID: "un_kg",
		UnitConversions: []domain.CreateUnitGroupUnitParams{
			{UnitID: "un_kg", DiscountPercentage: "0", DiscountFixed: "0", IsVisible: true},
		},
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Len(result.UnitConversions, 1)
}

func (suite *UnitGroupSvcTestSuite) TestCreateUnitGroup_TypeMismatch() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Mass Group TM", (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitQueryRepo.EXPECT().
		Find(gomock.Any(), "ac_test123", "un_vol").
		Return(&domain.LightUnit{ID: "un_vol", Type: "volume"}, nil).
		Times(1)
	suite.expectUGCacheError()

	result, err := suite.unitGroupSvc.CreateUnitGroup(ctx, domain.CreateUnitGroupParams{
		Name:       "Mass Group TM",
		Type:       "mass",
		BaseUnitID: "un_vol",
		UnitConversions: []domain.CreateUnitGroupUnitParams{
			{UnitID: "un_vol", DiscountPercentage: "0", DiscountFixed: "0", IsVisible: true},
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("unit_conversions", err.Param)
}

func (suite *UnitGroupSvcTestSuite) TestCreateUnitGroup_MissingIdentity() {
	result, err := suite.unitGroupSvc.CreateUnitGroup(context.Background(), domain.CreateUnitGroupParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestCreateUnitGroup_InsufficientPermissions() {
	ctx := readOnlyUnitGroupCtx("ac_test123")

	result, err := suite.unitGroupSvc.CreateUnitGroup(ctx, domain.CreateUnitGroupParams{Name: "Test"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestCreateUnitGroup_DuplicateName() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Existing Group", (*string)(nil)).
		Return(true, nil).
		Times(1)
	suite.expectUGCacheError()

	result, err := suite.unitGroupSvc.CreateUnitGroup(ctx, domain.CreateUnitGroupParams{
		Name:       "Existing Group",
		Type:       "mass",
		BaseUnitID: "un_base",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceConflict, err.Code)
	suite.Equal("name", err.Param)
}

func (suite *UnitGroupSvcTestSuite) TestCreateUnitGroup_IdempotencyFinished() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))
	code := 200
	accountID := "ac_test123"

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_ug_test",
			RecoveryPoint: string(domain.RecoveryPointFinished),
			ResponseCode:  &code,
			ResponseBody:  marshalJSON(&domain.UnitGroupFull{ID: "ug_cached", Name: "Mass Group", AccountID: &accountID}),
		}, nil).
		Times(1)

	result, err := suite.unitGroupSvc.CreateUnitGroup(ctx, domain.CreateUnitGroupParams{
		Name:       "Mass Group",
		Type:       "mass",
		BaseUnitID: "un_base",
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("ug_cached", result.ID)
}

func (suite *UnitGroupSvcTestSuite) TestCreateUnitGroup_UnexpectedRecoveryPoint() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_ug_test",
			RecoveryPoint: "unexpected_point",
		}, nil).
		Times(1)

	result, err := suite.unitGroupSvc.CreateUnitGroup(ctx, domain.CreateUnitGroupParams{
		Name:       "Mass Group",
		Type:       "mass",
		BaseUnitID: "un_base",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestCreateUnitGroup_RepoCreateError() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Mass Group Err", (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitQueryRepo.EXPECT().
		Find(gomock.Any(), "ac_test123", "un_base3").
		Return(&domain.LightUnit{ID: "un_base3", Type: "mass"}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("insert failed"), "db error")).
		Times(1)
	suite.expectUGCacheError()

	result, err := suite.unitGroupSvc.CreateUnitGroup(ctx, domain.CreateUnitGroupParams{
		Name:       "Mass Group Err",
		Type:       "mass",
		BaseUnitID: "un_base3",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

// --- UpdateUnitGroup ---

func (suite *UnitGroupSvcTestSuite) TestUpdateUnitGroup_Success() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))
	accountID := "ac_test123"
	newName := "Updated Group"

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_u1"}).
		Return(&domain.UnitGroupFull{ID: "ug_u1", Name: "Old Group", Type: "mass", AccountID: &accountID}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Updated Group", gomock.Any()).
		Return(false, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdateUnitGroupParams) (*domain.UnitGroupFull, *apierror.APIError) {
			suite.Equal("ug_u1", params.UnitGroupID)
			suite.Equal("Updated Group", *params.Name)
			return &domain.UnitGroupFull{ID: "ug_u1", Name: "Updated Group", AccountID: &accountID}, nil
		}).
		Times(1)
	suite.expectUGCacheSuccess()

	result, err := suite.unitGroupSvc.UpdateUnitGroup(ctx, domain.UpdateUnitGroupParams{
		UnitGroupID: "ug_u1",
		Name:        &newName,
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("Updated Group", result.Name)
}

func (suite *UnitGroupSvcTestSuite) TestUpdateUnitGroup_MissingIdentity() {
	result, err := suite.unitGroupSvc.UpdateUnitGroup(context.Background(), domain.UpdateUnitGroupParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestUpdateUnitGroup_InsufficientPermissions() {
	ctx := readOnlyUnitGroupCtx("ac_test123")

	result, err := suite.unitGroupSvc.UpdateUnitGroup(ctx, domain.UpdateUnitGroupParams{UnitGroupID: "ug_u2"})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestUpdateUnitGroup_SystemGroupRejected() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_system"}).
		Return(&domain.UnitGroupFull{ID: "ug_system", Name: "System Group", AccountID: nil}, nil).
		Times(1)
	suite.expectUGCacheError()

	result, err := suite.unitGroupSvc.UpdateUnitGroup(ctx, domain.UpdateUnitGroupParams{
		UnitGroupID: "ug_system",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "System unit groups cannot be modified")
}

func (suite *UnitGroupSvcTestSuite) TestUpdateUnitGroup_DuplicateName() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))
	accountID := "ac_test123"
	dupName := "Existing Group Name"

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_u3"}).
		Return(&domain.UnitGroupFull{ID: "ug_u3", Name: "Old Name", AccountID: &accountID}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Existing Group Name", gomock.Any()).
		Return(true, nil).
		Times(1)
	suite.expectUGCacheError()

	result, err := suite.unitGroupSvc.UpdateUnitGroup(ctx, domain.UpdateUnitGroupParams{
		UnitGroupID: "ug_u3",
		Name:        &dupName,
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceConflict, err.Code)
	suite.Equal("name", err.Param)
}

func (suite *UnitGroupSvcTestSuite) TestUpdateUnitGroup_NotFound() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_notfound2"}).
		Return(nil, apierror.NewResourceNotFoundError("Unit group not found.")).
		Times(1)
	suite.expectUGCacheError()

	result, err := suite.unitGroupSvc.UpdateUnitGroup(ctx, domain.UpdateUnitGroupParams{
		UnitGroupID: "ug_notfound2",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestUpdateUnitGroup_IdempotencyFinished() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))
	code := 200
	accountID := "ac_test123"

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_ug_test",
			RecoveryPoint: string(domain.RecoveryPointFinished),
			ResponseCode:  &code,
			ResponseBody:  marshalJSON(&domain.UnitGroupFull{ID: "ug_u_cached", Name: "Updated", AccountID: &accountID}),
		}, nil).
		Times(1)

	result, err := suite.unitGroupSvc.UpdateUnitGroup(ctx, domain.UpdateUnitGroupParams{
		UnitGroupID: "ug_u4",
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("ug_u_cached", result.ID)
}

func (suite *UnitGroupSvcTestSuite) TestUpdateUnitGroup_UnexpectedRecoveryPoint() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_ug_test",
			RecoveryPoint: "bad_point",
		}, nil).
		Times(1)

	result, err := suite.unitGroupSvc.UpdateUnitGroup(ctx, domain.UpdateUnitGroupParams{
		UnitGroupID: "ug_u5",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

// --- DeleteUnitGroup ---

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroup_Success() {
	ctx := internalUnitGroupCtx("ac_test123")
	accountID := "ac_test123"

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_d1"}).
		Return(&domain.UnitGroupFull{ID: "ug_d1", Name: "Delete Me", AccountID: &accountID}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		DeleteAllUnitGroupUnits(gomock.Any(), "ac_test123", "ug_d1").
		Return(nil).
		Times(1)
	suite.deletedRecordRepo.EXPECT().
		Create(gomock.Any(), constants.DeletedRecordResourceTypeUnitGroup, "ug_d1", gomock.Any()).
		Return(nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Delete(gomock.Any(), domain.DeleteUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_d1"}).
		Return(nil).
		Times(1)

	err := suite.unitGroupSvc.DeleteUnitGroup(ctx, "ug_d1")

	suite.Nil(err)
}

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroup_MissingIdentity() {
	err := suite.unitGroupSvc.DeleteUnitGroup(context.Background(), "ug_d2")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroup_InsufficientPermissions() {
	ctx := readOnlyUnitGroupCtx("ac_test123")

	err := suite.unitGroupSvc.DeleteUnitGroup(ctx, "ug_d3")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroup_SystemGroupRejected() {
	ctx := internalUnitGroupCtx("ac_test123")

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_sys"}).
		Return(&domain.UnitGroupFull{ID: "ug_sys", Name: "System Group", AccountID: nil}, nil).
		Times(1)

	err := suite.unitGroupSvc.DeleteUnitGroup(ctx, "ug_sys")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "System unit groups cannot be deleted")
}

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroup_AlreadyDeleted() {
	ctx := internalUnitGroupCtx("ac_test123")

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_gone"}).
		Return(nil, apierror.NewResourceNotFoundError("Unit group not found.")).
		Times(1)
	suite.deletedRecordRepo.EXPECT().
		Exists(gomock.Any(), constants.DeletedRecordResourceTypeUnitGroup, "ug_gone").
		Return(true, nil).
		Times(1)

	err := suite.unitGroupSvc.DeleteUnitGroup(ctx, "ug_gone")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceGone, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroup_NotFound() {
	ctx := internalUnitGroupCtx("ac_test123")

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_d_nf"}).
		Return(nil, apierror.NewResourceNotFoundError("Unit group not found.")).
		Times(1)
	suite.deletedRecordRepo.EXPECT().
		Exists(gomock.Any(), constants.DeletedRecordResourceTypeUnitGroup, "ug_d_nf").
		Return(false, nil).
		Times(1)

	err := suite.unitGroupSvc.DeleteUnitGroup(ctx, "ug_d_nf")

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

// --- UpsertUnitGroupUnit ---

func (suite *UnitGroupSvcTestSuite) TestUpsertUnitGroupUnit_MissingIdentity() {
	result, err := suite.unitGroupSvc.UpsertUnitGroupUnit(context.Background(), domain.UpsertUnitGroupUnitParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestUpsertUnitGroupUnit_InsufficientPermissions() {
	ctx := readOnlyUnitGroupCtx("ac_test123")

	result, err := suite.unitGroupSvc.UpsertUnitGroupUnit(ctx, domain.UpsertUnitGroupUnitParams{
		UnitGroupID: "ug_abc123",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestUpsertUnitGroupUnit_SystemGroupRejected() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_sys2"}).
		Return(&domain.UnitGroupFull{ID: "ug_sys2", Name: "System", AccountID: nil, Type: "mass"}, nil).
		Times(1)
	suite.expectUGCacheError()

	result, err := suite.unitGroupSvc.UpsertUnitGroupUnit(ctx, domain.UpsertUnitGroupUnitParams{
		UnitGroupID:        "ug_sys2",
		UnitID:             "un_abc",
		DiscountPercentage: "0",
		DiscountFixed:      "0",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "System unit groups cannot be modified")
}

func (suite *UnitGroupSvcTestSuite) TestUpsertUnitGroupUnit_TypeMismatch() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))
	accountID := "ac_test123"

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_mass1"}).
		Return(&domain.UnitGroupFull{
			ID:        "ug_mass1",
			Name:      "Mass Group",
			Type:      "mass",
			AccountID: &accountID,
		}, nil).
		Times(1)
	suite.unitQueryRepo.EXPECT().
		Find(gomock.Any(), "ac_test123", "un_vol").
		Return(&domain.LightUnit{ID: "un_vol", Type: "volume"}, nil).
		Times(1)
	suite.expectUGCacheError()

	result, err := suite.unitGroupSvc.UpsertUnitGroupUnit(ctx, domain.UpsertUnitGroupUnitParams{
		UnitGroupID:        "ug_mass1",
		UnitID:             "un_vol",
		DiscountPercentage: "0",
		DiscountFixed:      "0",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("unit_id", err.Param)
}

func (suite *UnitGroupSvcTestSuite) TestUpsertUnitGroupUnit_Success() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))
	accountID := "ac_test123"

	suite.expectUGIdempotencyStarted()
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_mass2"}).
		Return(&domain.UnitGroupFull{
			ID:        "ug_mass2",
			Name:      "Mass Group",
			Type:      "mass",
			AccountID: &accountID,
		}, nil).
		Times(1)
	suite.unitQueryRepo.EXPECT().
		Find(gomock.Any(), "ac_test123", "un_kg").
		Return(&domain.LightUnit{ID: "un_kg", Type: "mass"}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		UpsertUnitGroupUnit(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupUnit{ID: "uguID_kg", UnitID: "un_kg", UnitGroupID: "ug_mass2"}, nil).
		Times(1)
	suite.expectUGCacheSuccess()

	result, err := suite.unitGroupSvc.UpsertUnitGroupUnit(ctx, domain.UpsertUnitGroupUnitParams{
		UnitGroupID:        "ug_mass2",
		UnitID:             "un_kg",
		DiscountPercentage: "0",
		DiscountFixed:      "0",
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("un_kg", result.UnitID)
}

func (suite *UnitGroupSvcTestSuite) TestUpsertUnitGroupUnit_IdempotencyFinished() {
	ctx := idempotencyCtx(internalUnitGroupCtx("ac_test123"))
	code := 200

	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_ug_test",
			RecoveryPoint: string(domain.RecoveryPointFinished),
			ResponseCode:  &code,
			ResponseBody:  marshalJSON(&domain.UnitGroupUnit{ID: "ugu_cached", UnitID: "un_kg"}),
		}, nil).
		Times(1)

	result, err := suite.unitGroupSvc.UpsertUnitGroupUnit(ctx, domain.UpsertUnitGroupUnitParams{
		UnitGroupID: "ug_mass2",
		UnitID:      "un_kg",
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("ugu_cached", result.ID)
}

// --- DeleteUnitGroupUnit ---

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroupUnit_MissingIdentity() {
	err := suite.unitGroupSvc.DeleteUnitGroupUnit(context.Background(), domain.DeleteUnitGroupUnitParams{})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroupUnit_InsufficientPermissions() {
	ctx := readOnlyUnitGroupCtx("ac_test123")

	err := suite.unitGroupSvc.DeleteUnitGroupUnit(ctx, domain.DeleteUnitGroupUnitParams{
		UnitGroupID: "ug_abc123",
	})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroupUnit_SystemGroupRejected() {
	ctx := internalUnitGroupCtx("ac_test123")

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_sys3"}).
		Return(&domain.UnitGroupFull{ID: "ug_sys3", AccountID: nil}, nil).
		Times(1)

	err := suite.unitGroupSvc.DeleteUnitGroupUnit(ctx, domain.DeleteUnitGroupUnitParams{
		UnitGroupID:     "ug_sys3",
		UnitGroupUnitID: "uguID_x",
	})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "System unit groups cannot be modified")
}

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroupUnit_AlreadyDeleted() {
	ctx := internalUnitGroupCtx("ac_test123")
	accountID := "ac_test123"

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_d_u1"}).
		Return(&domain.UnitGroupFull{
			ID:              "ug_d_u1",
			AccountID:       &accountID,
			UnitConversions: []*domain.UnitGroupUnit{},
		}, nil).
		Times(1)
	suite.deletedRecordRepo.EXPECT().
		Exists(gomock.Any(), constants.DeletedRecordResourceTypeUnitGroupUnit, "uguID_gone").
		Return(true, nil).
		Times(1)

	err := suite.unitGroupSvc.DeleteUnitGroupUnit(ctx, domain.DeleteUnitGroupUnitParams{
		UnitGroupID:     "ug_d_u1",
		UnitGroupUnitID: "uguID_gone",
	})

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceGone, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestDeleteUnitGroupUnit_Success() {
	ctx := internalUnitGroupCtx("ac_test123")
	accountID := "ac_test123"
	existingUnit := &domain.UnitGroupUnit{
		ID:                 "uguID_del",
		UnitID:             "un_kg2",
		UnitGroupID:        "ug_d_u2",
		DiscountPercentage: "0",
		DiscountFixed:      "0",
	}

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_d_u2"}).
		Return(&domain.UnitGroupFull{
			ID:              "ug_d_u2",
			AccountID:       &accountID,
			UnitConversions: []*domain.UnitGroupUnit{existingUnit},
		}, nil).
		Times(1)
	suite.deletedRecordRepo.EXPECT().
		Create(gomock.Any(), constants.DeletedRecordResourceTypeUnitGroupUnit, "uguID_del", gomock.Any()).
		Return(nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		DeleteUnitGroupUnit(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	err := suite.unitGroupSvc.DeleteUnitGroupUnit(ctx, domain.DeleteUnitGroupUnitParams{
		UnitGroupID:     "ug_d_u2",
		UnitGroupUnitID: "uguID_del",
	})

	suite.Nil(err)
}

// --- ListUnitGroupUnits ---

func (suite *UnitGroupSvcTestSuite) TestListUnitGroupUnits_MissingIdentity() {
	result, err := suite.unitGroupSvc.ListUnitGroupUnits(context.Background(), "ug_abc123", nil)

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestListUnitGroupUnits_GroupNotFound() {
	ctx := internalUnitGroupCtx("ac_test123")

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_lu_nf"}).
		Return(nil, apierror.NewResourceNotFoundError("not found")).
		Times(1)

	result, err := suite.unitGroupSvc.ListUnitGroupUnits(ctx, "ug_lu_nf", nil)

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestListUnitGroupUnits_Success() {
	ctx := internalUnitGroupCtx("ac_test123")
	accountID := "ac_test123"

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_lu1"}).
		Return(&domain.UnitGroupFull{ID: "ug_lu1", AccountID: &accountID}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		ListUnits(gomock.Any(), "ug_lu1", gomock.Any()).
		Return([]*domain.UnitGroupUnit{
			{ID: "uguID_lu1", UnitID: "un_a", UnitGroupID: "ug_lu1"},
		}, nil).
		Times(1)

	result, err := suite.unitGroupSvc.ListUnitGroupUnits(ctx, "ug_lu1", nil)

	suite.Nil(err)
	suite.NotNil(result)
	suite.Len(result, 1)
	suite.Equal("un_a", result[0].UnitID)
}

// --- GetUnitGroupUnit ---

func (suite *UnitGroupSvcTestSuite) TestGetUnitGroupUnit_MissingIdentity() {
	result, err := suite.unitGroupSvc.GetUnitGroupUnit(context.Background(), domain.GetUnitGroupUnitParams{})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestGetUnitGroupUnit_GroupNotFound() {
	ctx := internalUnitGroupCtx("ac_test123")

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_gu_nf"}).
		Return(nil, apierror.NewResourceNotFoundError("not found")).
		Times(1)

	result, err := suite.unitGroupSvc.GetUnitGroupUnit(ctx, domain.GetUnitGroupUnitParams{
		UnitGroupID:     "ug_gu_nf",
		UnitGroupUnitID: "uguID_x",
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeResourceNotFound, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestGetUnitGroupUnit_Success() {
	ctx := internalUnitGroupCtx("ac_test123")
	accountID := "ac_test123"

	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: "ac_test123", UnitGroupID: "ug_gu1"}).
		Return(&domain.UnitGroupFull{ID: "ug_gu1", AccountID: &accountID}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		GetUnit(gomock.Any(), domain.GetUnitGroupUnitParams{
			AccountID:       "ac_test123",
			UnitGroupID:     "ug_gu1",
			UnitGroupUnitID: "uguID_gu1",
		}).
		Return(&domain.UnitGroupUnit{ID: "uguID_gu1", UnitID: "un_kg3", UnitGroupID: "ug_gu1"}, nil).
		Times(1)

	result, err := suite.unitGroupSvc.GetUnitGroupUnit(ctx, domain.GetUnitGroupUnitParams{
		UnitGroupID:     "ug_gu1",
		UnitGroupUnitID: "uguID_gu1",
	})

	suite.Nil(err)
	suite.NotNil(result)
	suite.Equal("uguID_gu1", result.ID)
}

// --- BulkUpsertUnitGroups ---

func (suite *UnitGroupSvcTestSuite) TestBulkUpsertUnitGroups_MissingIdentity() {
	result, err := suite.unitGroupSvc.BulkUpsertUnitGroups(context.Background(), domain.BulkUpsertUnitGroupsParams{
		UnitGroups: []domain.UpsertUnitGroupParams{{Name: "Test", Type: "mass", BaseUnit: domain.UnitIdentifier{ID: "un_base"}}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestBulkUpsertUnitGroups_InsufficientPermissions_NoCreate() {
	ctx := readOnlyUnitGroupCtx("ac_test123")

	result, err := suite.unitGroupSvc.BulkUpsertUnitGroups(ctx, domain.BulkUpsertUnitGroupsParams{
		UnitGroups: []domain.UpsertUnitGroupParams{{Name: "Test", Type: "mass", BaseUnit: domain.UnitIdentifier{ID: "un_base"}}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestBulkUpsertUnitGroups_InsufficientPermissions_NoUpdate() {
	ctx := createOnlyUnitGroupCtx("ac_test123")

	result, err := suite.unitGroupSvc.BulkUpsertUnitGroups(ctx, domain.BulkUpsertUnitGroupsParams{
		UnitGroups: []domain.UpsertUnitGroupParams{{Name: "Test", Type: "mass", BaseUnit: domain.UnitIdentifier{ID: "un_base"}}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestBulkUpsertUnitGroups_Empty() {
	ctx := internalUnitGroupCtx("ac_test123")

	result, err := suite.unitGroupSvc.BulkUpsertUnitGroups(ctx, domain.BulkUpsertUnitGroupsParams{
		UnitGroups: []domain.UpsertUnitGroupParams{},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "No unit groups provided")
}

func (suite *UnitGroupSvcTestSuite) TestBulkUpsertUnitGroups_TooMany() {
	ctx := internalUnitGroupCtx("ac_test123")
	groups := make([]domain.UpsertUnitGroupParams, 1001)

	result, err := suite.unitGroupSvc.BulkUpsertUnitGroups(ctx, domain.BulkUpsertUnitGroupsParams{
		UnitGroups: groups,
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "1000")
}

func (suite *UnitGroupSvcTestSuite) TestBulkUpsertUnitGroups_DuplicateNameInRequest() {
	ctx := internalUnitGroupCtx("ac_test123")

	result, err := suite.unitGroupSvc.BulkUpsertUnitGroups(ctx, domain.BulkUpsertUnitGroupsParams{
		UnitGroups: []domain.UpsertUnitGroupParams{
			{Name: "Mass Group", Type: "mass", BaseUnit: domain.UnitIdentifier{ID: "un_b1"}},
			{Name: "mass group", Type: "mass", BaseUnit: domain.UnitIdentifier{ID: "un_b2"}}, // same name case-insensitive
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("unit_groups[1].name", err.Param)
	suite.Contains(err.PublicMessage, "duplicate name")
}

// --- resolveBulkUpsertUnitGroupRows (the engine's Resolve hook, exercised directly) ---
//
// Resolve runs in the synchronous accept phase: it turns fuzzy unit references into ids
// (carrying each unit's dimension) and fails fast with a row-indexed 400 for an
// unresolvable reference or a unit listed twice in one group. It reads only units, so
// these tests mock the unit resolver — never the group repo (that is Write's).

// resolveUnitGroups runs the Resolve hook against the suite's mocked repo factory.
func (suite *UnitGroupSvcTestSuite) resolveUnitGroups(rows ...domain.UpsertUnitGroupParams) ([]domain.ResolvedUpsertUnitGroupRow, *apierror.APIError) {
	return resolveBulkUpsertUnitGroupRows(internalUnitGroupCtx("ac_test123"), suite.repoFactory, "ac_test123", rows)
}

// Duplicates are detected on the resolved unit id, so the same unit named twice in one
// group is rejected however it was referenced.
func (suite *UnitGroupSvcTestSuite) TestResolveBulkUpsertUnitGroups_DuplicateUnitInGroup() {
	suite.expectUnitsResolveByID("mass", nil)

	resolved, err := suite.resolveUnitGroups(domain.UpsertUnitGroupParams{
		Name:     "Mass Group Dup",
		Type:     "mass",
		BaseUnit: domain.UnitIdentifier{ID: "un_b3"},
		UnitConversions: []domain.UpsertUnitConversionParams{
			{Unit: domain.UnitIdentifier{ID: "un_kg4"}, DiscountPercentage: "0"},
			{Unit: domain.UnitIdentifier{ID: "un_kg4"}, DiscountPercentage: "0.1"}, // same unit twice
		},
	})

	suite.Nil(resolved)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "Duplicate unit")
	suite.Equal("unit_groups[0].unit_conversions[1].unit", err.Param)
}

// A unit reference that does not resolve fails against the base unit first, naming the
// offending row and field.
func (suite *UnitGroupSvcTestSuite) TestResolveBulkUpsertUnitGroups_UnitNotFound() {
	// "" → the unit is absent from the lookup, so the identifier does not resolve.
	suite.expectUnitsResolveByID("mass", map[string]string{"un_missing": ""})

	resolved, err := suite.resolveUnitGroups(domain.UpsertUnitGroupParams{
		Name:     "Mass Group Missing",
		Type:     "mass",
		BaseUnit: domain.UnitIdentifier{ID: "un_missing"},
		UnitConversions: []domain.UpsertUnitConversionParams{
			{Unit: domain.UnitIdentifier{ID: "un_missing"}, DiscountPercentage: "0"},
		},
	})

	suite.Nil(resolved)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Equal("unit_groups[0].base_unit", err.Param)
}

// --- writeBulkUpsertUnitGroups (the engine's Write hook, exercised directly) ---
//
// The accept phase and job plumbing are covered by the engine's own test; these rows
// prove the group-specific write logic: name matching, the create/update split, the
// deferred dimension check against a group's stored type, system-group rejection, and
// the base-unit auto-include. Write takes pre-resolved rows, so no resolver mock is set.

// writeUnitGroups runs the Write hook against the suite's mocked repo factory and returns
// the created/updated ids (from results) and the per-row failures (from errors). apiErr is
// non-nil only for a pre-loop infrastructure failure (the bulk read, a data invariant); a
// row that fails its own upsert is recorded in rowErrs, not returned as apiErr.
func (suite *UnitGroupSvcTestSuite) writeUnitGroups(rows ...domain.ResolvedUpsertUnitGroupRow) (created, updated []string, rowErrs []apierror.RowError, apiErr *apierror.APIError) {
	res, apiErr := writeBulkUpsertUnitGroups(internalUnitGroupCtx("ac_test123"), suite.repoFactory, passthroughSavepoint{}, "ac_test123", rows)
	if apiErr != nil {
		return nil, nil, nil, apiErr
	}
	created, updated = splitJobResults(res.Results)
	return created, updated, res.Errors, nil
}

// The dimension check needs the group's stored type, so it runs in Write per-row: a
// conversion of the wrong dimension is recorded in errors, not a synchronous 400.
func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_TypeMismatch() {
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.UnitGroupFull{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		FindUnitsByGroupIDs(gomock.Any(), gomock.Any()).
		Return([]*domain.UnitGroupUnit{}, nil).
		Times(1)

	created, updated, rowErrs, err := suite.writeUnitGroups(domain.ResolvedUpsertUnitGroupRow{
		Name:                  "Mass Group Mismatch",
		Type:                  "mass",
		BaseUnitID:            "un_b4",
		BaseUnitDimensionCode: "mass",
		Conversions: []domain.ResolvedUnitGroupConversion{
			{UnitID: "un_vol2", DimensionCode: "volume", DiscountPercentage: "0"},
		},
	})

	suite.Nil(err)
	suite.Empty(created)
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
	suite.Equal(0, rowErrIndex(rowErrs[0]))
	suite.Contains(rowErrMessage(rowErrs[0]), "does not match")
}

// The base unit is held to the same dimension rule as conversions: a base unit of the
// wrong dimension is recorded per-row against its own field, not the conversions'.
func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_BaseUnitTypeMismatch() {
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.UnitGroupFull{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		FindUnitsByGroupIDs(gomock.Any(), gomock.Any()).
		Return([]*domain.UnitGroupUnit{}, nil).
		Times(1)

	created, updated, rowErrs, err := suite.writeUnitGroups(domain.ResolvedUpsertUnitGroupRow{
		Name:                  "Mass Group Bad Base",
		Type:                  "mass",
		BaseUnitID:            "un_vol_base",
		BaseUnitDimensionCode: "volume", // wrong dimension for a mass group
	})

	suite.Nil(err)
	suite.Empty(created)
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
	suite.Equal(0, rowErrIndex(rowErrs[0]))
	suite.Contains(rowErrMessage(rowErrs[0]), "Base unit type does not match")
}

func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_FindByNamesError() {
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("db error"), "query failed")).
		Times(1)

	_, _, _, err := suite.writeUnitGroups(domain.ResolvedUpsertUnitGroupRow{Name: "Test BU4", Type: "mass", BaseUnitID: "un_b8"})

	suite.NotNil(err, "a failed bulk read is infrastructural — it fails the whole batch")
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_EmptyIDInExistingGroup() {
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.UnitGroupFull{{ID: "", Name: "Test BU5"}}, nil).
		Times(1)

	_, _, _, err := suite.writeUnitGroups(domain.ResolvedUpsertUnitGroupRow{Name: "Test BU5", Type: "mass", BaseUnitID: "un_b9"})

	suite.NotNil(err, "a corrupt existing row is an invariant violation, not a per-row failure")
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_AllCreates() {
	accountID := "ac_test123"
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.UnitGroupFull{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		FindUnitsByGroupIDs(gomock.Any(), gomock.Any()).
		Return([]*domain.UnitGroupUnit{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "New Group", (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_created1", Name: "New Group", AccountID: &accountID}, nil).
		Times(1)
	// The base unit is auto-included as an associated unit.
	suite.unitGroupRepo.EXPECT().
		UpsertUnitGroupUnit(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupUnit{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_created1", Name: "New Group", AccountID: &accountID}, nil).
		Times(1)

	created, updated, rowErrs, err := suite.writeUnitGroups(domain.ResolvedUpsertUnitGroupRow{
		Name:                  "New Group",
		Type:                  "mass",
		BaseUnitID:            "un_base_c",
		BaseUnitDimensionCode: "mass",
	})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 1)
	suite.Equal("ug_created1", created[0])
	suite.Nil(updated)
}

func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_AllUpdates() {
	accountID := "ac_test123"
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.UnitGroupFull{{ID: "ug_existing1", Name: "Existing Group", Type: "mass", AccountID: &accountID}}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		FindUnitsByGroupIDs(gomock.Any(), gomock.Any()).
		Return([]*domain.UnitGroupUnit{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_existing1", Name: "Existing Group", AccountID: &accountID}, nil).
		Times(1)
	// The base unit is auto-included as an associated unit.
	suite.unitGroupRepo.EXPECT().
		UpsertUnitGroupUnit(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupUnit{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_existing1", Name: "Existing Group", AccountID: &accountID}, nil).
		Times(1)

	created, updated, rowErrs, err := suite.writeUnitGroups(domain.ResolvedUpsertUnitGroupRow{
		Name:                  "Existing Group",
		Type:                  "mass",
		BaseUnitID:            "un_base_u",
		BaseUnitDimensionCode: "mass",
	})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(updated, 1)
	suite.Equal("ug_existing1", updated[0])
	suite.Nil(created)
}

// An omitted note must reach the repository as unset, not as an explicit null: the update
// only writes the column when the field was provided, so Clear here would wipe the note.
func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_OmittedNotesLeavesTheStoredOne() {
	accountID := "ac_test123"
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.UnitGroupFull{{ID: "ug_notes", Name: "Notes Group", Type: "mass", AccountID: &accountID}}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		FindUnitsByGroupIDs(gomock.Any(), gomock.Any()).
		Return([]*domain.UnitGroupUnit{}, nil).
		Times(1)

	var captured domain.UpdateUnitGroupParams
	suite.unitGroupRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdateUnitGroupParams) (*domain.UnitGroupFull, *apierror.APIError) {
			captured = params
			return &domain.UnitGroupFull{ID: "ug_notes", Name: "Notes Group", AccountID: &accountID}, nil
		}).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		UpsertUnitGroupUnit(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupUnit{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_notes", Name: "Notes Group", AccountID: &accountID}, nil).
		Times(1)

	_, updated, rowErrs, err := suite.writeUnitGroups(domain.ResolvedUpsertUnitGroupRow{
		Name:                  "Notes Group",
		Type:                  "mass",
		BaseUnitID:            "un_base_n",
		BaseUnitDimensionCode: "mass",
	})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(updated, 1)
	suite.False(captured.Notes.WasProvided(), "an omitted note must not be written")
	suite.False(captured.Notes.IsClear(), "an omitted note must not null the column")
}

func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_MixedCreateAndUpdate() {
	accountID := "ac_test123"
	// "Existing Group Mix" exists; "New Group Mix" does not
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.UnitGroupFull{{ID: "ug_mix_existing", Name: "Existing Group Mix", Type: "mass", AccountID: &accountID}}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		FindUnitsByGroupIDs(gomock.Any(), gomock.Any()).
		Return([]*domain.UnitGroupUnit{}, nil).
		Times(1)
	// Create for new group
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "New Group Mix", (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_mix_new", Name: "New Group Mix", AccountID: &accountID}, nil).
		Times(1)
	// Update for existing group
	suite.unitGroupRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_mix_existing", Name: "Existing Group Mix", AccountID: &accountID}, nil).
		Times(1)
	// The base unit is auto-included as an associated unit (once per group).
	suite.unitGroupRepo.EXPECT().
		UpsertUnitGroupUnit(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupUnit{}, nil).
		Times(2)
	// Get called once per group (create path and update path each re-fetch)
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.GetUnitGroupParams) (*domain.UnitGroupFull, *apierror.APIError) {
			return &domain.UnitGroupFull{ID: params.UnitGroupID, AccountID: &accountID}, nil
		}).
		Times(2)

	created, updated, rowErrs, err := suite.writeUnitGroups(
		domain.ResolvedUpsertUnitGroupRow{Name: "New Group Mix", Type: "mass", BaseUnitID: "un_base_m1", BaseUnitDimensionCode: "mass"},
		domain.ResolvedUpsertUnitGroupRow{Name: "Existing Group Mix", Type: "mass", BaseUnitID: "un_base_m2", BaseUnitDimensionCode: "mass"},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 1)
	suite.Len(updated, 1)
}

// A system group (nil AccountID) cannot be modified: the row is rejected per-row and
// recorded in errors, not returned as a batch error.
func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_SystemGroupRejected() {
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		// AccountID nil → system group; cannot be modified.
		Return([]*domain.UnitGroupFull{{ID: "ug_sys_bu", Name: "System Group BU", Type: "mass", AccountID: nil}}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		FindUnitsByGroupIDs(gomock.Any(), gomock.Any()).
		Return([]*domain.UnitGroupUnit{}, nil).
		Times(1)

	created, updated, rowErrs, err := suite.writeUnitGroups(domain.ResolvedUpsertUnitGroupRow{
		Name:                  "System Group BU",
		Type:                  "mass",
		BaseUnitID:            "un_base_s",
		BaseUnitDimensionCode: "mass",
	})

	suite.Nil(err)
	suite.Empty(created)
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
	suite.Contains(rowErrMessage(rowErrs[0]), "cannot be modified")
}

func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_CreateWithConversions() {
	accountID := "ac_test123"
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.UnitGroupFull{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		FindUnitsByGroupIDs(gomock.Any(), gomock.Any()).
		Return([]*domain.UnitGroupUnit{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Conv Group", (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_ai1", Name: "Conv Group", AccountID: &accountID}, nil).
		Times(1)

	var capturedUpsertParams domain.UpsertUnitGroupUnitParams
	// Base unit == the single conversion unit, so the auto-include de-dupes to one upsert.
	suite.unitGroupRepo.EXPECT().
		UpsertUnitGroupUnit(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params domain.UpsertUnitGroupUnitParams) (*domain.UnitGroupUnit, *apierror.APIError) {
			capturedUpsertParams = params
			return &domain.UnitGroupUnit{ID: "uguID_ai", UnitID: params.UnitID}, nil
		}).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_ai1", Name: "Conv Group", AccountID: &accountID}, nil).
		Times(1)

	created, _, rowErrs, err := suite.writeUnitGroups(domain.ResolvedUpsertUnitGroupRow{
		Name:                  "Conv Group",
		Type:                  "mass",
		BaseUnitID:            "un_base_ai",
		BaseUnitDimensionCode: "mass",
		Conversions: []domain.ResolvedUnitGroupConversion{
			{UnitID: "un_base_ai", DimensionCode: "mass", DiscountPercentage: "0"},
		},
	})

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 1)
	suite.Equal("un_base_ai", capturedUpsertParams.UnitID)
	suite.Equal("0", capturedUpsertParams.DiscountPercentage)
}

// Partial success: a valid create writes while a bad row (wrong-dimension conversion) is
// recorded in errors — the write does not fail the batch.
func (suite *UnitGroupSvcTestSuite) TestWriteBulkUpsertUnitGroups_PartialSuccess() {
	accountID := "ac_test123"
	suite.unitGroupRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.UnitGroupFull{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		FindUnitsByGroupIDs(gomock.Any(), gomock.Any()).
		Return([]*domain.UnitGroupUnit{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		ExistsByName(gomock.Any(), "ac_test123", "Good Group", (*string)(nil)).
		Return(false, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_good", Name: "Good Group", AccountID: &accountID}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		UpsertUnitGroupUnit(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupUnit{}, nil).
		Times(1)
	suite.unitGroupRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(&domain.UnitGroupFull{ID: "ug_good", Name: "Good Group", AccountID: &accountID}, nil).
		Times(1)

	// Row 0 creates "Good Group"; row 1 has a volume conversion in a mass group.
	created, updated, rowErrs, err := suite.writeUnitGroups(
		domain.ResolvedUpsertUnitGroupRow{Name: "Good Group", Type: "mass", BaseUnitID: "un_base_g", BaseUnitDimensionCode: "mass"},
		domain.ResolvedUpsertUnitGroupRow{
			Name:                  "Bad Group",
			Type:                  "mass",
			BaseUnitID:            "un_base_b",
			BaseUnitDimensionCode: "mass",
			Conversions: []domain.ResolvedUnitGroupConversion{
				{UnitID: "un_vol", DimensionCode: "volume", DiscountPercentage: "0"},
			},
		},
	)

	suite.Nil(err)
	suite.Len(created, 1, "the valid row still writes")
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
	suite.Equal(1, rowErrIndex(rowErrs[0]))
	suite.Contains(rowErrMessage(rowErrs[0]), "does not match")
}

// --- test helper ---

func marshalJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// reopens the export so assertions read what a spreadsheet would
func (suite *UnitGroupSvcTestSuite) exportedRows(export *domain.Export) [][]string {
	return exportedSheetRows(suite.T(), export, "Unit Groups")
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *UnitGroupSvcTestSuite) buildExport(ctx context.Context, params domain.ExportUnitGroupsParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.unitGroupSvc.(*unitGroupSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}

// the group's own columns sit on its first unit's row and are blank on the rest
func (suite *UnitGroupSvcTestSuite) TestExportUnitGroups_ListsUnitsOnePerRow() {
	ctx := internalUnitGroupCtx("ac_test123")
	notes := "Weights"
	suite.unitGroupRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.UnitGroupFull{
		{
			ID:       "ung_1",
			Name:     "Mass",
			Type:     "mass",
			Notes:    &notes,
			BaseUnit: domain.LightUnit{ID: "unt_kg", Name: "Kilogram", Abbreviation: "kg"},
			UnitConversions: []*domain.UnitGroupUnit{
				// The base unit is filtered out: it already has its own column.
				{UnitID: "unt_kg", DiscountPercentage: "0", Unit: domain.LightUnit{Name: "Kilogram", Abbreviation: "kg"}},
				{UnitID: "unt_g", DiscountPercentage: "0.5", Unit: domain.LightUnit{Name: "Gram", Abbreviation: "g"}},
				{UnitID: "unt_t", DiscountPercentage: "0", Unit: domain.LightUnit{Name: "Tonne", Abbreviation: "t"}},
			},
		},
	}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportUnitGroupsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(1), export.RowCount, "one resource row, even though it spans two sheet rows")

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 3)
	suite.Equal([]string{"ID", "Name", "Type", "Base Unit", "Units", "Discount %'s", "Notes"}, rows[0])
	suite.Equal([]string{"ung_1", "Mass", "mass", "Kilogram, kg", "Gram, g", "50", "Weights"}, rows[1])
	// The continuation row carries only the unit; the group's columns are blank.
	suite.Equal([]string{"", "", "", "", "Tonne, t"}, rows[2])
}

// a group whose only conversion is its base unit still gets a row
func (suite *UnitGroupSvcTestSuite) TestExportUnitGroups_GroupWithNoOtherUnitsStillExports() {
	ctx := internalUnitGroupCtx("ac_test123")
	suite.unitGroupRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.UnitGroupFull{
		{
			ID:       "ung_2",
			Name:     "Each",
			Type:     "quantity",
			BaseUnit: domain.LightUnit{ID: "unt_ea", Name: "Each", Abbreviation: "ea"},
			UnitConversions: []*domain.UnitGroupUnit{
				{UnitID: "unt_ea", Unit: domain.LightUnit{Name: "Each", Abbreviation: "ea"}},
			},
		},
	}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportUnitGroupsParams{})
	suite.Require().Nil(apiErr)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 2)
	suite.Equal([]string{"ung_2", "Each", "quantity", "Each, ea"}, rows[1])
}

func (suite *UnitGroupSvcTestSuite) TestExportUnitGroups_ScopesToTheIdentitysAccount() {
	ctx := internalUnitGroupCtx("ac_owner")
	suite.unitGroupRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportUnitGroupsParams) ([]*domain.UnitGroupFull, error) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportUnitGroupsParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}

func TestDiscountPercent(t *testing.T) {
	tests := []struct {
		name   string
		stored string
		want   string
	}{
		{name: "half off", stored: "0.5", want: "50"},
		{name: "trims trailing zeros", stored: "0.125", want: "12.5"},
		{name: "zero stays blank so re-import keeps the default", stored: "0", want: ""},
		{name: "unparseable stays blank", stored: "", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, discountPercent(tc.stored))
		})
	}
}
