package apikeyep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type APIKeySvc interface {
	GetAPIKey(ctx context.Context, req *GetAPIKeyRequest) (*apiresource.APIKey, *apierror.APIError)
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

func (m *apiKeySvcImpl) GetAPIKey(ctx context.Context, req *GetAPIKeyRequest) (*apiresource.APIKey, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, apiKeySvcTracer, "service.api_keys.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAPIKeyResponse, error) {
			return m.authClient.GetAPIKey(ctx, &pb.GetAPIKeyRequest{
				ApiKeyId: req.APIKeyID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	presented := APIKeyPresenter(resp.ApiKey)
	return &presented, nil
}

func (m *apiKeySvcImpl) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*apiresource.CreatedAPIKey, *apierror.APIError) {
	pbReq := &pb.CreateAPIKeyRequest{
		RoleId: req.RoleID,
		Name:   req.Name,
	}

	if req.ExpiresAt != nil {
		pbReq.ExpiresAt = timestamppb.New(*req.ExpiresAt)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, apiKeySvcTracer, "service.api_keys.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAPIKeyResponse, error) {
			return m.authClient.CreateAPIKey(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	presented := APIKeyCreatedPresenter(resp)
	return &presented, nil
}

func (m *apiKeySvcImpl) RotateAPIKey(ctx context.Context, req *RotateAPIKeyRequest) (*apiresource.CreatedAPIKey, *apierror.APIError) {
	pbReq := &pb.RotateAPIKeyRequest{
		ApiKeyId: req.APIKeyID,
	}

	if req.ExpiresAt != nil {
		pbReq.ExpiresAt = timestamppb.New(*req.ExpiresAt)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, apiKeySvcTracer, "service.api_keys.rotate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.RotateAPIKeyResponse, error) {
			return m.authClient.RotateAPIKey(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	presented := APIKeyRotatedPresenter(resp)
	return &presented, nil
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
	statuses := make([]string, len(req.Status))
	for i, s := range req.Status {
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

	return APIKeyListPresenter(resp), nil
}

func (m *apiKeySvcImpl) GetOrCreateDocAPIKey(ctx context.Context) (*apiresource.CreatedAPIKey, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, apiKeySvcTracer, "service.api_keys.get_or_create_doc", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetOrCreateDocAPIKeyResponse, error) {
			return m.authClient.GetOrCreateDocAPIKey(ctx, &pb.GetOrCreateDocAPIKeyRequest{}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	presented := APIKeyDocPresenter(resp)
	return &presented, nil
}
