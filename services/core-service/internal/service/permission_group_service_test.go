package service

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type PermissionGroupSvcTestSuite struct {
	suite.Suite
	permissionGroupSvc  domain.PermissionGroupSvc
	permissionGroupRepo *repositorymock.MockPermissionGroupRepo
	repoFactory         *factorymock.MockRepoFactory
	ctrl                *gomock.Controller
}

func (suite *PermissionGroupSvcTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.permissionGroupRepo = repositorymock.NewMockPermissionGroupRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewPermissionGroupRepo().Return(suite.permissionGroupRepo).AnyTimes()

	suite.permissionGroupSvc = NewPermissionGroupSvc(&PermissionGroupSvcConfig{
		Repos: suite.repoFactory,
	})
}

func (suite *PermissionGroupSvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestPermissionGroupSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PermissionGroupSvcTestSuite))
}

func permissionGroupInternalIdentityCtx() context.Context {
	adminCode := string(constants.RoleTypeCodeAdmin)
	targetAccountID := "acct_test"
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &targetAccountID,
			RoleTypeCode: &adminCode,
			Permissions: map[string]bool{
				"permissions:read": true,
			},
		},
	})
}

func (suite *PermissionGroupSvcTestSuite) TestListPermissionGroups_Success() {
	ctx := permissionGroupInternalIdentityCtx()

	expected := &domain.ListPermissionGroupsResult{
		PermissionGroups: []*domain.PermissionGroup{
			{
				ID:   "pg_test1",
				Code: "customers",
				Name: "Customers",
				Permissions: []*domain.Permission{
					{
						ID:                  "perm_test1",
						Code:                "customers:read",
						Name:                "Read Customers",
						PermissionGroupCode: "customers",
						CreatedAt:           time.Now(),
						UpdatedAt:           time.Now(),
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		PageInfo: pagination.PageInfo{HasNextPage: false, HasPrevPage: false},
	}

	params := domain.ListPermissionGroupsParams{
		Limit: 100,
	}

	suite.permissionGroupRepo.EXPECT().List(gomock.Any(), params).Return(expected, nil)

	result, apiErr := suite.permissionGroupSvc.ListPermissionGroups(ctx, params)
	suite.Nil(apiErr)
	suite.NotNil(result)
	suite.Len(result.PermissionGroups, 1)
	suite.Equal("pg_test1", result.PermissionGroups[0].ID)
	suite.Len(result.PermissionGroups[0].Permissions, 1)
}

func (suite *PermissionGroupSvcTestSuite) TestListPermissionGroups_RequiresInternalActor() {
	// Customer actor should be rejected
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeCustomer,
			ID:           "usr_customer",
			Permissions: map[string]bool{
				"permissions:read": true,
			},
		},
	})

	params := domain.ListPermissionGroupsParams{Limit: 100}

	result, apiErr := suite.permissionGroupSvc.ListPermissionGroups(ctx, params)
	suite.Nil(result)
	suite.NotNil(apiErr)
}

func (suite *PermissionGroupSvcTestSuite) TestListPermissionGroups_RequiresPermission() {
	// Internal actor with custom role, without permissions:read
	customCode := string(constants.RoleTypeCodeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			RoleTypeCode: &customCode,
			Permissions:  map[string]bool{},
		},
	})

	params := domain.ListPermissionGroupsParams{Limit: 100}

	result, apiErr := suite.permissionGroupSvc.ListPermissionGroups(ctx, params)
	suite.Nil(result)
	suite.NotNil(apiErr)
}

func (suite *PermissionGroupSvcTestSuite) TestListPermissionGroups_NoIdentity() {
	ctx := context.Background()

	params := domain.ListPermissionGroupsParams{Limit: 100}

	result, apiErr := suite.permissionGroupSvc.ListPermissionGroups(ctx, params)
	suite.Nil(result)
	suite.NotNil(apiErr)
}
