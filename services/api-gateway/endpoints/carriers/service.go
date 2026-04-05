package carrierep

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
	"google.golang.org/protobuf/types/known/emptypb"
)

type CarrierSvc interface {
	ListCarriers(ctx context.Context, req *ListCarriersRequest) (*apiresource.List[apiresource.Carrier], *apierror.APIError)
	GetCarrier(ctx context.Context, req *GetCarrierRequest) (*apiresource.Carrier, *apierror.APIError)
	CreateCarrier(ctx context.Context, req *CreateCarrierRequest) (*apiresource.Carrier, *apierror.APIError)
	UpdateCarrier(ctx context.Context, req *UpdateCarrierRequest) (*apiresource.Carrier, *apierror.APIError)
	DeleteCarrier(ctx context.Context, req *DeleteCarrierRequest) (*apiresource.EmptyResource, *apierror.APIError)
	InitiateOAuth(ctx context.Context, req *InitiateOAuthRequest) (*apiresource.OAuthResponse, *apierror.APIError)
	GetOAuthStatus(ctx context.Context, req *GetOAuthStatusRequest) (*apiresource.OAuthStatusResponse, *apierror.APIError)
	SyncOptions(ctx context.Context, req *SyncOptionsRequest) (*apiresource.Carrier, *apierror.APIError)
}

type CarrierSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type carrierSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var carrierSvcTracer = tracing.GetTracer("api-gateway.endpoints.carriers.service")

func (c *CarrierSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("carrier endpoint service: core client is required")
	}
	return nil
}

func NewCarrierSvc(config *CarrierSvcConfig) CarrierSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &carrierSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *carrierSvcImpl) ListCarriers(ctx context.Context, req *ListCarriersRequest) (*apiresource.List[apiresource.Carrier], *apierror.APIError) {
	pbReq := &pb.ListCarriersRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCarriersResponse, error) {
			return m.coreClient.ListCarriers(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return CarrierListPresenter(resp), nil
}

func (m *carrierSvcImpl) GetCarrier(ctx context.Context, req *GetCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
	pbReq := &pb.GetCarrierRequest{
		Id: req.CarrierID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCarrierResponse, error) {
			return m.coreClient.GetCarrier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CarrierPresenter(resp.Carrier)
	return &result, nil
}

func (m *carrierSvcImpl) CreateCarrier(ctx context.Context, req *CreateCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
	isPortalEnabled := true
	if req.CustomerPortalVisibility != nil {
		isPortalEnabled = *req.CustomerPortalVisibility == constants.CustomerPortalVisibilityVisible
	}

	var code *string
	if req.Code != nil {
		s := string(*req.Code)
		code = &s
	}

	pbReq := &pb.CreateCarrierRequest{
		Name:            req.Name,
		Code:            code,
		AccountNumber:   req.AccountNumber,
		IsPortalEnabled: isPortalEnabled,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateCarrierResponse, error) {
			return m.coreClient.CreateCarrier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CarrierPresenter(resp.Carrier)
	return &result, nil
}

func (m *carrierSvcImpl) UpdateCarrier(ctx context.Context, req *UpdateCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
	var isPortalEnabled *bool
	if req.CustomerPortalVisibility != nil {
		v := *req.CustomerPortalVisibility == constants.CustomerPortalVisibilityVisible
		isPortalEnabled = &v
	}

	pbReq := &pb.UpdateCarrierRequest{
		Id:              req.CarrierID,
		Name:            req.Name,
		IsPortalEnabled: isPortalEnabled,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateCarrierResponse, error) {
			return m.coreClient.UpdateCarrier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CarrierPresenter(resp.Carrier)
	return &result, nil
}

func (m *carrierSvcImpl) DeleteCarrier(ctx context.Context, req *DeleteCarrierRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteCarrierRequest{
		Id: req.CarrierID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteCarrier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *carrierSvcImpl) InitiateOAuth(ctx context.Context, req *InitiateOAuthRequest) (*apiresource.OAuthResponse, *apierror.APIError) {
	pbReq := &pb.InitiateCarrierOAuthRequest{
		CarrierId:   req.CarrierID,
		RedirectUri: req.RedirectURI,
		State:       req.State,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.initiate_oauth", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.InitiateCarrierOAuthResponse, error) {
			return m.coreClient.InitiateCarrierOAuth(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.OAuthResponse{Object: constants.ObjectTypeOAuthResponse, OAuthURL: resp.OauthUrl}, nil
}

func (m *carrierSvcImpl) GetOAuthStatus(ctx context.Context, req *GetOAuthStatusRequest) (*apiresource.OAuthStatusResponse, *apierror.APIError) {
	pbReq := &pb.GetCarrierOAuthStatusRequest{
		CarrierId: req.CarrierID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.get_oauth_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCarrierOAuthStatusResponse, error) {
			return m.coreClient.GetCarrierOAuthStatus(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.OAuthStatusResponse{Object: constants.ObjectTypeOAuthStatusResponse, Status: resp.Status}, nil
}

func (m *carrierSvcImpl) SyncOptions(ctx context.Context, req *SyncOptionsRequest) (*apiresource.Carrier, *apierror.APIError) {
	pbReq := &pb.SyncServiceLevelsRequest{
		CarrierId: req.CarrierID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, carrierSvcTracer, "service.carriers.sync_options", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SyncServiceLevelsResponse, error) {
			return m.coreClient.SyncServiceLevels(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CarrierPresenter(resp.Carrier)
	return &result, nil
}
