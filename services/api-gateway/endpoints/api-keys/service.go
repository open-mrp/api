package apikeyep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type APIKeySvc interface {
	GetAPIKey(ctx context.Context, req *RetrieveAPIKeyRequest) (*apiresource.APIKey, *apierror.APIError)
	CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*apiresource.CreatedAPIKey, *apierror.APIError)
	RotateAPIKey(ctx context.Context, req *RotateAPIKeyRequest) (*apiresource.CreatedAPIKey, *apierror.APIError)
	RevokeAPIKey(ctx context.Context, req *RevokeAPIKeyRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListAPIKeys(ctx context.Context, req *ListAPIKeysRequest) (*apiresource.List[apiresource.APIKey], *apierror.APIError)
	GetOrCreateDocAPIKey(ctx context.Context) (*apiresource.CreatedAPIKey, *apierror.APIError)
}

type APIKeySvcConfig struct {
	AuthClient pb.AuthServiceClient
}

type apiKeySvcImpl struct {
	authClient pb.AuthServiceClient
}

var apiKeySvcTracer = tracing.GetTracer("api-gateway.endpoints.api_keys.service")

func (c *APIKeySvcConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("api key endpoint service: auth client is required")
	}
	return nil
}

func NewAPIKeySvc(config *APIKeySvcConfig) APIKeySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &apiKeySvcImpl{
		authClient: config.AuthClient,
	}
}

func loadAPIKeyByID(ctx context.Context, id string) (*apiresource.APIKey, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadAPIKeys(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("API key not found.")
	}
	return v.(*apiresource.APIKey), nil
}

func (m *apiKeySvcImpl) GetAPIKey(ctx context.Context, req *RetrieveAPIKeyRequest) (*apiresource.APIKey, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, apiKeySvcTracer, "service.api_keys.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAPIKeyResponse, error) {
			return m.authClient.GetAPIKey(ctx, &pb.GetAPIKeyRequest{
				ApiKeyId: req.APIKeyID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadAPIKeyByID(ctx, resp.ApiKey.Id)
}

func (m *apiKeySvcImpl) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*apiresource.CreatedAPIKey, *apierror.APIError) {
	pbReq := &pb.CreateAPIKeyRequest{
		RoleId: req.RoleID,
		Name:   req.Name,
	}

	if v, ok := req.ExpiresAt.Value(); ok {
		pbReq.ExpiresAt = timestamppb.New(v)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, apiKeySvcTracer, "service.api_keys.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAPIKeyResponse, error) {
			return m.authClient.CreateAPIKey(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	loaded, apiErr := loadAPIKeyByID(ctx, resp.ApiKey.Id)
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.CreatedAPIKey{
		Object:       constants.ObjectTypeCreatedAPIKey,
		APIKeySecret: resp.ApiKeySecret,
		APIKeyInfo:   *loaded,
	}, nil
}

func (m *apiKeySvcImpl) RotateAPIKey(ctx context.Context, req *RotateAPIKeyRequest) (*apiresource.CreatedAPIKey, *apierror.APIError) {
	pbReq := &pb.RotateAPIKeyRequest{
		ApiKeyId: req.APIKeyID,
	}

	if v, ok := req.ExpiresAt.Value(); ok {
		pbReq.ExpiresAt = timestamppb.New(v)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, apiKeySvcTracer, "service.api_keys.rotate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.RotateAPIKeyResponse, error) {
			return m.authClient.RotateAPIKey(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	loaded, apiErr := loadAPIKeyByID(ctx, resp.ApiKey.Id)
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.CreatedAPIKey{
		Object:       constants.ObjectTypeCreatedAPIKey,
		APIKeySecret: resp.ApiKeySecret,
		APIKeyInfo:   *loaded,
	}, nil
}

func (m *apiKeySvcImpl) RevokeAPIKey(ctx context.Context, req *RevokeAPIKeyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, apiKeySvcTracer, "service.api_keys.revoke", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.authClient.RevokeAPIKey(ctx, &pb.RevokeAPIKeyRequest{
				ApiKeyId: req.APIKeyID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *apiKeySvcImpl) ListAPIKeys(ctx context.Context, req *ListAPIKeysRequest) (*apiresource.List[apiresource.APIKey], *apierror.APIError) {
	statuses := make([]string, len(req.Statuses))
	for i, s := range req.Statuses {
		statuses[i] = string(s)
	}

	pbReq := &pb.ListAPIKeysRequest{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Statuses: statuses,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, apiKeySvcTracer, "service.api_keys.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAPIKeysResponse, error) {
			return m.authClient.ListAPIKeys(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.ApiKeys))
	for i, k := range resp.ApiKeys {
		ids[i] = k.Id
	}

	loaded, apiErr := resourceloaders.LoadAPIKeys(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}

	keys := make([]apiresource.APIKey, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			keys = append(keys, *v.(*apiresource.APIKey))
		}
	}

	return apiresource.NewList(keys, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *apiKeySvcImpl) GetOrCreateDocAPIKey(ctx context.Context) (*apiresource.CreatedAPIKey, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, apiKeySvcTracer, "service.api_keys.get_or_create_doc", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetOrCreateDocAPIKeyResponse, error) {
			return m.authClient.GetOrCreateDocAPIKey(ctx, &pb.GetOrCreateDocAPIKeyRequest{}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	loaded, apiErr := loadAPIKeyByID(ctx, resp.ApiKey.Id)
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.CreatedAPIKey{
		Object:       constants.ObjectTypeCreatedAPIKey,
		APIKeySecret: resp.ApiKeySecret,
		APIKeyInfo:   *loaded,
	}, nil
}
