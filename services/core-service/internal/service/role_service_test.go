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
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type RoleSvcTestSuite struct {
	suite.Suite
	roleSvc            domain.RoleSvc
	roleRepo           *repositorymock.MockRoleRepo
	rolePermissionRepo *repositorymock.MockRolePermissionRepo
	accountUserRepo    *repositorymock.MockAccountUserRepo
	deletedRecordRepo  *repositorymock.MockDeletedRecordRepo
	repoFactory        *factorymock.MockRepoFactory
	idempotencyMed     *mediatormock.MockIdempotencyMed
	ctrl               *gomock.Controller
}

type roleTestMediatorFactory struct {
	mediators domain.Mediators
}

func (f *roleTestMediatorFactory) Build(_ domain.RepoFactory) domain.Mediators {
	return f.mediators
}

func (suite *RoleSvcTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.roleRepo = repositorymock.NewMockRoleRepo(suite.ctrl)
	suite.rolePermissionRepo = repositorymock.NewMockRolePermissionRepo(suite.ctrl)
	suite.accountUserRepo = repositorymock.NewMockAccountUserRepo(suite.ctrl)
	suite.deletedRecordRepo = repositorymock.NewMockDeletedRecordRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewRoleRepo().Return(suite.roleRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewRolePermissionRepo().Return(suite.rolePermissionRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAccountUserRepo().Return(suite.accountUserRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewDeletedRecordRepo().Return(suite.deletedRecordRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	mediatorFactory := &roleTestMediatorFactory{mediators: domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}}

	suite.roleSvc = NewRoleSvc(&RoleSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *RoleSvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestRoleSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RoleSvcTestSuite))
}

func roleIdentityCtx(targetAccountID string) context.Context {
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
				"roles:read":   true,
				"roles:create": true,
				"roles:update": true,
				"roles:delete": true,
			},
		},
	})
}

func roleIdempotencyCtx(ctx context.Context) context.Context {
	ctx = appctx.WithIdempotencyKey(ctx, "test-idempotency-key")
	ctx = appctx.WithHandler(ctx, "/core.CoreService/CreateRole")
	ctx = appctx.WithIdempotencyResponseMetadata(ctx, &appctx.IdempotencyResponseMetadata{})
	return ctx
}

func (suite *RoleSvcTestSuite) expectRoleIdempotencyStarted() {
	suite.idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_test",
			RecoveryPoint: string(domain.RecoveryPointStarted),
		}, nil).
		Times(1)
}

// TestListRoles_Success verifies that list roles works with proper permissions.
func (suite *RoleSvcTestSuite) TestListRoles_Success() {
	ctx := roleIdentityCtx("acct_test")

	suite.roleRepo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(&domain.ListRolesPage{
			Roles: []*domain.Role{
				{ID: "rl_1", Name: "Admin", RoleType: "admin"},
			},
		}, nil).
		Times(1)

	suite.rolePermissionRepo.EXPECT().
		ListByRoleIDs(gomock.Any(), []string{"rl_1"}).
		Return(map[string][]*domain.RolePermission{}, nil).
		Times(1)

	result, apiErr := suite.roleSvc.ListRoles(ctx, domain.ListRolesParams{
		Limit:    10,
		Includes: []string{"permissions"},
	})
	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Len(result.Roles, 1)
}

// TestListRoles_RequiresReadPermission verifies that the create permission is rejected.
func (suite *RoleSvcTestSuite) TestListRoles_RequiresReadPermission() {
	customCode := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "acct_test"},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			RoleType:     &customCode,
			Permissions:  map[string]bool{},
		},
	})

	_, apiErr := suite.roleSvc.ListRoles(ctx, domain.ListRolesParams{
		Limit: 10,
	})
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

// TestGetRole_Success verifies that get role returns role with permissions.
func (suite *RoleSvcTestSuite) TestGetRole_Success() {
	ctx := roleIdentityCtx("acct_test")
	accountID := "acct_test"

	suite.roleRepo.EXPECT().
		Get(gomock.Any(), "rl_1", "acct_test").
		Return(&domain.Role{
			ID:        "rl_1",
			Name:      "Manager",
			RoleType:  "user",
			AccountID: &accountID,
		}, nil).
		Times(1)

	suite.rolePermissionRepo.EXPECT().
		ListByRoleID(gomock.Any(), "rl_1").
		Return([]*domain.RolePermission{
			{ID: "rlpm_1", PermissionCode: "customers", Create: true, Read: true},
		}, nil).
		Times(1)

	result, apiErr := suite.roleSvc.GetRole(ctx, "rl_1", []string{"permissions"})
	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Equal("rl_1", result.ID)
	suite.Len(result.Permissions, 1)
	suite.Equal("customers", result.Permissions[0].PermissionCode)
}

// TestCreateRole_DuplicateName verifies that creating a role with a duplicate name fails.
func (suite *RoleSvcTestSuite) TestCreateRole_DuplicateName() {
	ctx := roleIdempotencyCtx(roleIdentityCtx("acct_test"))

	suite.expectRoleIdempotencyStarted()
	suite.roleRepo.EXPECT().
		ExistsByName(gomock.Any(), "acct_test", "Manager", nil).
		Return(true, nil).
		Times(1)

	suite.idempotencyMed.EXPECT().
		CacheErrorResponse(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		}).
		Times(1)

	_, apiErr := suite.roleSvc.CreateRole(ctx, domain.CreateRoleParams{
		Name: "Manager",
	})
	suite.NotNil(apiErr)
}

// TestDeleteRole_CannotDeleteGlobal verifies that global roles cannot be deleted.
func (suite *RoleSvcTestSuite) TestDeleteRole_CannotDeleteGlobal() {
	ctx := roleIdentityCtx("acct_test")

	suite.roleRepo.EXPECT().
		Get(gomock.Any(), "rl_global", "acct_test").
		Return(&domain.Role{
			ID:        "rl_global",
			Name:      "Admin",
			RoleType:  "admin",
			AccountID: nil,
		}, nil).
		Times(1)

	apiErr := suite.roleSvc.DeleteRole(ctx, "rl_global")
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeValidationFailed, apiErr.Code)
}

// TestDeleteRole_Success verifies that account-owned roles can be deleted.
func (suite *RoleSvcTestSuite) TestDeleteRole_Success() {
	ctx := roleIdentityCtx("acct_test")
	accountID := "acct_test"

	suite.roleRepo.EXPECT().
		Get(gomock.Any(), "rl_1", "acct_test").
		Return(&domain.Role{
			ID:        "rl_1",
			Name:      "Manager",
			RoleType:  "user",
			AccountID: &accountID,
		}, nil).
		Times(1)

	suite.accountUserRepo.EXPECT().
		CountByRoleID(gomock.Any(), "acct_test", "rl_1").
		Return(int64(0), nil).
		Times(1)

	suite.deletedRecordRepo.EXPECT().
		Create(gomock.Any(), constants.DeletedRecordResourceTypeRole, "rl_1", gomock.Any()).
		Return(nil).
		Times(1)

	suite.rolePermissionRepo.EXPECT().
		DeleteByRoleID(gomock.Any(), "rl_1").
		Return(nil).
		Times(1)

	suite.roleRepo.EXPECT().
		Delete(gomock.Any(), "rl_1", "acct_test").
		Return(nil).
		Times(1)

	apiErr := suite.roleSvc.DeleteRole(ctx, "rl_1")
	suite.Nil(apiErr)
}

// TestDeleteRole_BlockedWhenAssigned verifies delete is blocked when role has assigned users.
func (suite *RoleSvcTestSuite) TestDeleteRole_BlockedWhenAssigned() {
	ctx := roleIdentityCtx("acct_test")
	accountID := "acct_test"

	suite.roleRepo.EXPECT().
		Get(gomock.Any(), "rl_1", "acct_test").
		Return(&domain.Role{
			ID:        "rl_1",
			Name:      "Manager",
			RoleType:  "user",
			AccountID: &accountID,
		}, nil).
		Times(1)

	suite.accountUserRepo.EXPECT().
		CountByRoleID(gomock.Any(), "acct_test", "rl_1").
		Return(int64(1), nil).
		Times(1)

	apiErr := suite.roleSvc.DeleteRole(ctx, "rl_1")
	suite.NotNil(apiErr)
	suite.Equal(apierror.ErrorCodeResourceConflict, apiErr.Code)
}
