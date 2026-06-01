package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	authpb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var apiKeyLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.api_key")

func LoadAPIKeys(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, apiKeyLoaderTracer, "loader.api_keys.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*authpb.BatchGetAPIKeysByIDsResponse, error) {
			return authClient.BatchGetAPIKeysByIDs(ctx, &authpb.BatchGetAPIKeysByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.ApiKeys))
	for _, k := range resp.ApiKeys {
		out[k.Id] = apiKeyFromProto(k)
		if k.RoleId != nil {
			meta.Set(constants.ObjectTypeAPIKey, k.Id, "role_id", *k.RoleId)
		}
	}
	return out, nil
}

func apiKeyFromProto(k *authpb.APIKeyInfo) *apiresource.APIKey {
	return &apiresource.APIKey{
		ID:            k.Id,
		Object:        constants.ObjectTypeAPIKey,
		Name:          k.Name,
		RedactedValue: k.RedactedValue,
		CreatedAt:     grpcutil.TimestampToTime(k.CreatedAt),
		UpdatedAt:     grpcutil.TimestampToTime(k.UpdatedAt),
		LastUsedAt:    grpcutil.TimestampToTimePtr(k.LastUsedAt),
		ExpiresAt:     grpcutil.TimestampToTimePtr(k.ExpiresAt),
		RevokedAt:     grpcutil.TimestampToTimePtr(k.RevokedAt),
	}
}
