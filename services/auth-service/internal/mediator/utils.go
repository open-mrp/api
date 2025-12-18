package mediator

import (
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
)

func buildUnassignedAPIKeyIdentity(apiKeyModel *domain.APIKey) *types.Identity {
	return &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: nil,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeUnassigned,
			ID:           apiKeyModel.ID,
			Name:         &apiKeyModel.Name,
			AccountID:    &apiKeyModel.OwnerAccountID,
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
	}
}

func buildOwnedAPIKeyIdentity(apiKeyModel *domain.APIKey, targetAccountID string, permissions map[string]bool) *types.Identity {
	roleTypeCode := apiKeyModel.RoleTypeCode

	return &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: &targetAccountID,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           apiKeyModel.ID,
			Name:         &apiKeyModel.Name,
			AccountID:    &apiKeyModel.OwnerAccountID,
			RoleID:       &apiKeyModel.RoleID,
			RoleTypeCode: &roleTypeCode,
			Permissions:  permissions,
		},
		AccountMode: constants.AccountModeProduction,
	}
}

func buildRelatedAPIKeyIdentity(apiKeyModel *domain.APIKey, accountRelation *domain.AuthAccountRelation, actorType types.IdentityActorType, targetAccountID string) *types.Identity {
	return &types.Identity{
		Type:            types.IdentityTypeAPIKey,
		TargetAccountID: &targetAccountID,
		Actor: &types.IdentityActor{
			Type:         actorType,
			ID:           apiKeyModel.ID,
			Name:         &apiKeyModel.Name,
			AccountID:    &accountRelation.CounterpartyAccountID,
			RoleID:       nil,
			RoleTypeCode: nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
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

func buildRelatedUserIdentity(userModel *types.User, accountRelation *domain.AuthAccountRelation, actorType types.IdentityActorType, targetAccountID string) *types.Identity {
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
		AccountMode: constants.AccountModeProduction,
	}
}

func buildAccountUserIdentity(userModel *types.User, accountUser *domain.AccountUser, permissions map[string]bool, targetAccountID string) *types.Identity {
	return &types.Identity{
		Type:            types.IdentityTypeUser,
		TargetAccountID: &targetAccountID,
		Actor: &types.IdentityActor{
			Type:         types.IdentityActorTypeInternal,
			ID:           userModel.ID,
			Name:         userModel.Name,
			AccountID:    &accountUser.AccountID,
			RoleID:       accountUser.RoleID,
			RoleTypeCode: accountUser.RoleTypeCode,
			Permissions:  permissions,
		},
		AccountMode: constants.AccountModeProduction,
	}
}
