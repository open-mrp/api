package mediator

import (
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
)

func buildOwnedAPIKeyIdentity(apiKeyModel *domain.APIKey, targetAccountID string, permissions map[string]bool, accountMode constants.AccountMode) *types.Identity {
	roleTypeCode := apiKeyModel.RoleTypeCode

	return &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: &targetAccountID,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           apiKeyModel.TypeID,
			Name:         &apiKeyModel.Name,
			AccountID:    &apiKeyModel.OwnerAccountID,
			RoleID:       &apiKeyModel.RoleID,
			RoleTypeCode: &roleTypeCode,
			Permissions:  permissions,
		},
		AccountMode: accountMode,
	}
}

func buildRelatedAPIKeyIdentity(apiKeyModel *domain.APIKey, accountRelation *domain.AuthAccountRelation, actorType types.IdentityActorType, targetAccountID string, accountMode constants.AccountMode) *types.Identity {
	return &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: &targetAccountID,
		Actor: &types.IdentityActor{
			Type:         actorType,
			ID:           apiKeyModel.TypeID,
			Name:         &apiKeyModel.Name,
			AccountID:    &accountRelation.CounterpartyAccountID,
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: accountMode,
	}
}

func buildUnassignedUserIdentity(userModel *types.User) *types.Identity {
	return &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: nil,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeUnassigned,
			ID:           userModel.ID,
			Name:         userModel.Name,
			AccountID:    nil,
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
	}
}

func buildRelatedUserIdentity(userModel *types.User, accountRelation *domain.AuthAccountRelation, actorType types.IdentityActorType, targetAccountID string, accountMode constants.AccountMode) *types.Identity {
	return &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: &targetAccountID,
		Actor: &types.IdentityActor{
			Type:         actorType,
			ID:           userModel.ID,
			Name:         userModel.Name,
			AccountID:    &accountRelation.CounterpartyAccountID,
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: accountMode,
	}
}

func buildAccountUserIdentity(userModel *types.User, access *domain.AccountUserAccess, targetAccountID string, accountMode constants.AccountMode) *types.Identity {
	return &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: &targetAccountID,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           userModel.ID,
			Name:         userModel.Name,
			AccountID:    &access.AccountID,
			RoleID:       access.RoleID,
			RoleTypeCode: access.RoleTypeCode,
			Permissions:  access.Permissions,
		},
		AccountMode: accountMode,
	}
}
