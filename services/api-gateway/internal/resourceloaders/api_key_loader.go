package resourceloaders

import (
	"context"
	"time"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	authpb "github.com/open-mrp/api/shared/proto/auth"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
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

	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.ApiKeys))
	for _, k := range resp.ApiKeys {
		out[k.Id] = apiKeyFromProto(k)
		if k.RoleId != nil {
			meta.Set(constants.ObjectTypeAPIKey, k.Id, "role_id", *k.RoleId)
		}
	}
	return out, nil
}

// APIKeyFromProto builds the APIKey resource from the proto and stashes the role FK in LoadMeta so ?include=role resolves. List presenters use this directly so the response comes from the list RPC's single snapshot instead of a second batch read that can observe concurrent mutations.
func APIKeyFromProto(ctx context.Context, k *authpb.APIKeyInfo) *apiresource.APIKey {
	if k.RoleId != nil {
		include.GetLoadMeta(ctx).Set(constants.ObjectTypeAPIKey, k.Id, "role_id", *k.RoleId)
	}
	return apiKeyFromProto(k)
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
		Status:        apiKeyStatus(k, time.Now()),
		Client:        apiKeyClient(k),
	}
}

// apiKeyStatus is revoked once the revocation has taken effect, else expired once past its expiry, else active.
func apiKeyStatus(k *authpb.APIKeyInfo, now time.Time) constants.APIKeyStatus {
	if k.RevokedAt != nil && !k.RevokedAt.AsTime().After(now) {
		return constants.APIKeyStatusRevoked
	}
	if k.ExpiresAt != nil && !k.ExpiresAt.AsTime().After(now) {
		return constants.APIKeyStatusExpired
	}
	return constants.APIKeyStatusActive
}

func apiKeyClient(k *authpb.APIKeyInfo) *apiresource.APIKeyClient {
	if k.ClientName == nil || *k.ClientName == "" {
		return nil
	}
	return &apiresource.APIKeyClient{Name: *k.ClientName, Source: k.ClientSource}
}
