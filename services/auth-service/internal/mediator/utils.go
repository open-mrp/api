package mediator

import (
	"github.com/augno/api/services/auth-service/internal/apikey"
	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
)

func buildOwnedAPIKeyIdentity(apiKeyModel *apikey.APIKey, targetAccountID string, permissions map[string]bool, accountMode constants.AccountMode, subscriptionStatus *string) *types.Identity {
	return buildOwnedAPIKeyIdentityWithRelation(apiKeyModel, targetAccountID, permissions, accountMode, subscriptionStatus, nil)
}

func buildOwnedAPIKeyIdentityWithRelation(apiKeyModel *apikey.APIKey, targetAccountID string, permissions map[string]bool, accountMode constants.AccountMode, subscriptionStatus *string, targetRelationType *types.IdentityRelationType) *types.Identity {
	roleTypeCode := apiKeyModel.RoleType

	return &types.Identity{
		Type: types.IdentityActorTypeAPIKey,
		Target: &types.IdentityTarget{
			AccountID:    targetAccountID,
			RelationType: targetRelationType,
		},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           apiKeyModel.TypeID,
			Name:         &apiKeyModel.Name,
			AccountID:    &apiKeyModel.OwnerAccountID,
			RoleID:       &apiKeyModel.RoleID,
			RoleType:     &roleTypeCode,
			Permissions:  permissions,
		},
		AccountMode:        accountMode,
		SubscriptionStatus: subscriptionStatus,
	}
}

func buildRelatedAPIKeyIdentity(apiKeyModel *apikey.APIKey, accountRelation *domain.AuthAccountRelation, actorType types.IdentityRelationType, targetAccountID string, permissions map[string]bool, accountMode constants.AccountMode, subscriptionStatus *string) *types.Identity {
	relationType := accountRelation.AccountRelationRoleCode
	if permissions == nil {
		permissions = map[string]bool{}
	}
	return &types.Identity{
		Type: types.IdentityActorTypeAPIKey,
		Target: &types.IdentityTarget{
			AccountID:    targetAccountID,
			RelationType: &relationType,
		},
		Actor: &types.IdentityActor{
			RelationType: actorType,
			ID:           apiKeyModel.TypeID,
			Name:         &apiKeyModel.Name,
			AccountID:    &accountRelation.CounterpartyAccountID,
			// RoleID/RoleType cleared so IsAdmin()/IsRoleSet() stay false; only the
			// carried own-account permissions authorize customer/supplier-side actions.
			RoleID:      nil,
			RoleType:    nil,
			Permissions: permissions,
		},
		AccountMode:        accountMode,
		SubscriptionStatus: subscriptionStatus,
	}
}

func buildUnassignedUserIdentity(userModel *types.User) *types.Identity {
	return &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeUnassigned,
			ID:           userModel.ID,
			Name:         userModel.Name,
			AccountID:    nil,
			RoleID:       nil,
			RoleType:     nil,
			Permissions:  map[string]bool{},
		},
		AccountMode: constants.AccountModeProduction,
	}
}

func buildRelatedUserIdentity(userModel *types.User, accountRelation *domain.AuthAccountRelation, actorType types.IdentityRelationType, targetAccountID string, accountMode constants.AccountMode, subscriptionStatus *string) *types.Identity {
	return &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: actorType,
			ID:           userModel.ID,
			Name:         userModel.Name,
			AccountID:    &accountRelation.CounterpartyAccountID,
			RoleID:       nil,
			RoleType:     nil,
			Permissions:  map[string]bool{},
		},
		AccountMode:        accountMode,
		SubscriptionStatus: subscriptionStatus,
	}
}

func buildAccountUserIdentity(userModel *types.User, access *domain.AccountUserAccess, targetAccountID string, accountMode constants.AccountMode, subscriptionStatus *string) *types.Identity {
	return &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           userModel.ID,
			Name:         userModel.Name,
			AccountID:    &access.AccountID,
			RoleID:       access.RoleID,
			RoleType:     access.RoleType,
			RoleName:     access.RoleName,
			Permissions:  access.Permissions,
		},
		AccountMode:        accountMode,
		SubscriptionStatus: subscriptionStatus,
	}
}
