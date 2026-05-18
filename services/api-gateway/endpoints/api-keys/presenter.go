package apikeyep

import (
	"context"
	"sort"

	"github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/auth"
)

func APIKeyPresenter(key *pb.APIKeyInfo, permissions map[string]bool) apiresource.APIKey {
	if key == nil {
		return apiresource.APIKey{}
	}

	res := apiresource.APIKey{
		ID:            key.Id,
		Object:        constants.ObjectTypeAPIKey,
		Name:          key.Name,
		RedactedValue: key.RedactedValue,
		CreatedAt:     grpc.TimestampToTime(key.CreatedAt),
		UpdatedAt:     grpc.TimestampToTime(key.UpdatedAt),
		LastUsedAt:    grpc.TimestampToTimePtr(key.LastUsedAt),
		ExpiresAt:     grpc.TimestampToTimePtr(key.ExpiresAt),
		RevokedAt:     grpc.TimestampToTimePtr(key.RevokedAt),
	}

	if key.RoleId != nil && key.RoleName != nil && key.RoleTypeCode != nil {
		res.Role = &apiresource.Role{
			ID:       *key.RoleId,
			Object:   constants.ObjectTypeRole,
			Name:     *key.RoleName,
			TypeCode: constants.RoleType(*key.RoleTypeCode),
			Owner:    apiresource.SystemOwner(),
		}
		if permissions != nil {
			perms := make([]string, 0, len(permissions))
			for p := range permissions {
				perms = append(perms, p)
			}
			sort.Strings(perms)
			res.Role.Permissions = &perms
		}
	}

	return res
}

func APIKeyCreatedPresenter(resp *pb.CreateAPIKeyResponse, permissions map[string]bool) apiresource.CreatedAPIKey {
	if resp == nil {
		return apiresource.CreatedAPIKey{}
	}

	return apiresource.CreatedAPIKey{
		Object:       constants.ObjectTypeCreatedAPIKey,
		APIKeySecret: resp.ApiKeySecret,
		APIKeyInfo:   APIKeyPresenter(resp.ApiKey, permissions),
	}
}

func APIKeyRotatedPresenter(resp *pb.RotateAPIKeyResponse, permissions map[string]bool) apiresource.CreatedAPIKey {
	if resp == nil {
		return apiresource.CreatedAPIKey{}
	}

	return apiresource.CreatedAPIKey{
		Object:       constants.ObjectTypeCreatedAPIKey,
		APIKeySecret: resp.ApiKeySecret,
		APIKeyInfo:   APIKeyPresenter(resp.ApiKey, permissions),
	}
}

func APIKeyDocPresenter(resp *pb.GetOrCreateDocAPIKeyResponse, permissions map[string]bool) apiresource.CreatedAPIKey {
	if resp == nil {
		return apiresource.CreatedAPIKey{}
	}

	return apiresource.CreatedAPIKey{
		Object:       constants.ObjectTypeCreatedAPIKey,
		APIKeySecret: resp.ApiKeySecret,
		APIKeyInfo:   APIKeyPresenter(resp.ApiKey, permissions),
	}
}

func APIKeyListPresenter(ctx context.Context, resp *pb.ListAPIKeysResponse, permResolver func(roleID *string) map[string]bool) *apiresource.List[apiresource.APIKey] {
	if resp == nil {
		return apiresource.NewList[apiresource.APIKey](nil, apiresource.PageInfo{})
	}

	keys := make([]apiresource.APIKey, len(resp.ApiKeys))
	for i, pbKey := range resp.ApiKeys {
		keys[i] = APIKeyPresenter(pbKey, permResolver(pbKey.RoleId))
	}

	return apiresource.NewList(keys, grpc.MapProtoPageInfo(ctx, resp.PageInfo))
}
