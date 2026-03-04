package apikeyep

import (
	"github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/auth"
)

func APIKeyPresenter(key *pb.APIKeyInfo) apiresource.APIKey {
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

	if key.RoleId != nil && key.RoleName != nil {
		role := &apiresource.LightRole{
			ID:     *key.RoleId,
			Object: constants.ObjectTypeRole,
			Name:   *key.RoleName,
		}
		if key.RoleTypeCode != nil {
			role.RoleTypeCode = constants.RoleTypeCode(*key.RoleTypeCode)
		}
		res.Role = role
	}

	return res
}

func APIKeyCreatedPresenter(resp *pb.CreateAPIKeyResponse) apiresource.CreatedAPIKey {
	if resp == nil {
		return apiresource.CreatedAPIKey{}
	}

	return apiresource.CreatedAPIKey{
		APIKeySecret: resp.ApiKeySecret,
		APIKeyInfo:   APIKeyPresenter(resp.ApiKey),
	}
}

func APIKeyRotatedPresenter(resp *pb.RotateAPIKeyResponse) apiresource.CreatedAPIKey {
	if resp == nil {
		return apiresource.CreatedAPIKey{}
	}

	return apiresource.CreatedAPIKey{
		APIKeySecret: resp.ApiKeySecret,
		APIKeyInfo:   APIKeyPresenter(resp.ApiKey),
	}
}

func APIKeyDocPresenter(resp *pb.GetOrCreateDocAPIKeyResponse) apiresource.CreatedAPIKey {
	if resp == nil {
		return apiresource.CreatedAPIKey{}
	}

	return apiresource.CreatedAPIKey{
		APIKeySecret: resp.ApiKeySecret,
		APIKeyInfo:   APIKeyPresenter(resp.ApiKey),
	}
}

func APIKeyListPresenter(resp *pb.ListAPIKeysResponse) *apiresource.List[apiresource.APIKey] {
	if resp == nil {
		return apiresource.NewList[apiresource.APIKey](nil, apiresource.PageInfo{})
	}

	keys := make([]apiresource.APIKey, len(resp.ApiKeys))
	for i, pbKey := range resp.ApiKeys {
		keys[i] = APIKeyPresenter(pbKey)
	}

	return apiresource.NewList(keys, mapProtoPageInfo(resp.PageInfo))
}

func mapProtoPageInfo(pi *pb.PageInfo) apiresource.PageInfo {
	if pi == nil {
		return apiresource.PageInfo{}
	}
	return apiresource.PageInfo{
		NextCursor:  pi.NextCursor,
		PrevCursor:  pi.PrevCursor,
		HasNextPage: pi.HasNextPage,
		HasPrevPage: pi.HasPrevPage,
	}
}
