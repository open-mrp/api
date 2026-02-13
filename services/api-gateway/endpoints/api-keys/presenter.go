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

	return apiresource.APIKey{
		ID:            key.Id,
		Object:        constants.ObjectTypeAPIKey,
		Name:          key.Name,
		RedactedValue: key.RedactedValue,
		Role: apiresource.LightRole{
			ID:   key.RoleId,
			Name: key.RoleName,
		},
		CreatedAt:  grpc.TimestampToTime(key.CreatedAt),
		UpdatedAt:  grpc.TimestampToTime(key.UpdatedAt),
		LastUsedAt: grpc.TimestampToTimePtr(key.LastUsedAt),
		ExpiresAt:  grpc.TimestampToTimePtr(key.ExpiresAt),
		RevokedAt:  grpc.TimestampToTimePtr(key.RevokedAt),
	}
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
		return apiresource.NewList[apiresource.APIKey](nil, false, nil)
	}

	keys := make([]apiresource.APIKey, len(resp.ApiKeys))
	for i, pbKey := range resp.ApiKeys {
		keys[i] = APIKeyPresenter(pbKey)
	}

	return apiresource.NewList(keys, resp.HasMore, resp.NextCursor)
}
