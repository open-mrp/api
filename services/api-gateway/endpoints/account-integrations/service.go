package accountintegrationep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type AccountIntegrationSvc interface {
	ListAccountIntegrations(ctx context.Context, req *ListAccountIntegrationsRequest) (*apiresource.List[apiresource.AccountIntegration], *apierror.APIError)
	CreateAccountIntegration(ctx context.Context, req *CreateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError)
	UpdateAccountIntegration(ctx context.Context, req *UpdateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError)
	DeleteAccountIntegration(ctx context.Context, req *DeleteAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError)
	GetStripePublishableKey(ctx context.Context, req *GetStripePublishableKeyRequest) (*apiresource.StripePublishableKey, *apierror.APIError)
	GetStripeStatus(ctx context.Context, req *GetStripeStatusRequest) (*apiresource.StripeStatus, *apierror.APIError)
}

type AccountIntegrationSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type accountIntegrationSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var accountIntegrationSvcTracer = tracing.GetTracer("api-gateway.endpoints.account-integrations.service")

func (c *AccountIntegrationSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account integration endpoint service: core client is required")
	}
	return nil
}

func NewAccountIntegrationSvc(config *AccountIntegrationSvcConfig) AccountIntegrationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountIntegrationSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *accountIntegrationSvcImpl) ListAccountIntegrations(ctx context.Context, req *ListAccountIntegrationsRequest) (*apiresource.List[apiresource.AccountIntegration], *apierror.APIError) {
	pbReq := &pb.ListAccountIntegrationsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountIntegrationSvcTracer, "service.account-integrations.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAccountIntegrationsResponse, error) {
			return m.coreClient.ListAccountIntegrations(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AccountIntegrationListPresenter(ctx, resp), nil
}

func (m *accountIntegrationSvcImpl) CreateAccountIntegration(ctx context.Context, req *CreateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
	pbReq := &pb.CreateAccountIntegrationRequest{
		Name:            req.Name,
		IntegrationCode: string(req.IntegrationCode),
		Credentials:     req.Credentials,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountIntegrationSvcTracer, "service.account-integrations.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAccountIntegrationResponse, error) {
			return m.coreClient.CreateAccountIntegration(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountIntegrationPresenter(resp.AccountIntegration)
	return &result, nil
}

func (m *accountIntegrationSvcImpl) UpdateAccountIntegration(ctx context.Context, req *UpdateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
	pbReq := &pb.UpdateAccountIntegrationRequest{
		Id:       req.AccountIntegrationID,
		Name:     req.Name,
		IsActive: req.IsActive,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountIntegrationSvcTracer, "service.account-integrations.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAccountIntegrationResponse, error) {
			return m.coreClient.UpdateAccountIntegration(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountIntegrationPresenter(resp.AccountIntegration)
	return &result, nil
}

func (m *accountIntegrationSvcImpl) DeleteAccountIntegration(ctx context.Context, req *DeleteAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
	pbReq := &pb.DeleteAccountIntegrationRequest{
		Id: req.AccountIntegrationID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountIntegrationSvcTracer, "service.account-integrations.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteAccountIntegrationResponse, error) {
			return m.coreClient.DeleteAccountIntegration(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountIntegrationPresenter(resp.AccountIntegration)
	return &result, nil
}

func (m *accountIntegrationSvcImpl) GetStripePublishableKey(ctx context.Context, req *GetStripePublishableKeyRequest) (*apiresource.StripePublishableKey, *apierror.APIError) {
	pbReq := &pb.GetStripePublishableKeyRequest{}

	resp, apiErr := grpcutil.CallRPC(ctx, accountIntegrationSvcTracer, "service.account-integrations.get-stripe-publishable-key", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetStripePublishableKeyResponse, error) {
			return m.coreClient.GetStripePublishableKey(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.StripePublishableKey{
		Object:         constants.ObjectTypeStripePublishableKey,
		PublishableKey: resp.PublishableKey,
	}, nil
}

func (m *accountIntegrationSvcImpl) GetStripeStatus(ctx context.Context, req *GetStripeStatusRequest) (*apiresource.StripeStatus, *apierror.APIError) {
	pbReq := &pb.GetStripeStatusRequest{}

	resp, apiErr := grpcutil.CallRPC(ctx, accountIntegrationSvcTracer, "service.account-integrations.get-stripe-status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetStripeStatusResponse, error) {
			return m.coreClient.GetStripeStatus(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.StripeStatus{
		Object:               constants.ObjectTypeStripeStatus,
		HasStripeIntegration: resp.HasStripeIntegration,
	}, nil
}
