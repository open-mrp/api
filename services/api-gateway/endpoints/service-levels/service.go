package servicelevelep

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

type ServiceLevelSvc interface {
	ListServiceLevels(ctx context.Context, req *ListServiceLevelsRequest) (*apiresource.List[apiresource.ServiceLevel], *apierror.APIError)
	GetServiceLevel(ctx context.Context, req *GetServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError)
	CreateServiceLevel(ctx context.Context, req *CreateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError)
	UpdateServiceLevel(ctx context.Context, req *UpdateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError)
	DeleteServiceLevel(ctx context.Context, req *DeleteServiceLevelRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ServiceLevelSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type serviceLevelSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var serviceLevelSvcTracer = tracing.GetTracer("api-gateway.endpoints.service_levels.service")

func (c *ServiceLevelSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("service level endpoint service: core client is required")
	}
	return nil
}

func NewServiceLevelSvc(config *ServiceLevelSvcConfig) ServiceLevelSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &serviceLevelSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *serviceLevelSvcImpl) ListServiceLevels(ctx context.Context, req *ListServiceLevelsRequest) (*apiresource.List[apiresource.ServiceLevel], *apierror.APIError) {
	pbReq := &pb.ListServiceLevelsRequest{
		CarrierId: req.CarrierID,
		Cursor:    req.Cursor,
		Limit:     req.Limit,
		Query:     req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, serviceLevelSvcTracer, "service.service_levels.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListServiceLevelsResponse, error) {
			return m.coreClient.ListServiceLevels(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ServiceLevelListPresenter(resp), nil
}

func (m *serviceLevelSvcImpl) GetServiceLevel(ctx context.Context, req *GetServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
	pbReq := &pb.GetServiceLevelRequest{
		CarrierId: req.CarrierID,
		Id:        req.ServiceLevelID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, serviceLevelSvcTracer, "service.service_levels.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetServiceLevelResponse, error) {
			return m.coreClient.GetServiceLevel(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ServiceLevelPresenter(resp.ServiceLevel)
	return &result, nil
}

func (m *serviceLevelSvcImpl) CreateServiceLevel(ctx context.Context, req *CreateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
	isPortalEnabled := true
	if req.CustomerPortalVisibility != nil {
		isPortalEnabled = *req.CustomerPortalVisibility == constants.CustomerPortalVisibilityVisible
	}

	pbReq := &pb.CreateServiceLevelRequest{
		CarrierId:       req.CarrierID,
		Name:            req.Name,
		Code:            req.Code,
		IsPortalEnabled: isPortalEnabled,
		IsDefault:       req.IsDefault,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, serviceLevelSvcTracer, "service.service_levels.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateServiceLevelResponse, error) {
			return m.coreClient.CreateServiceLevel(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ServiceLevelPresenter(resp.ServiceLevel)
	return &result, nil
}

func (m *serviceLevelSvcImpl) UpdateServiceLevel(ctx context.Context, req *UpdateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
	var isPortalEnabled *bool
	if req.CustomerPortalVisibility != nil {
		v := *req.CustomerPortalVisibility == constants.CustomerPortalVisibilityVisible
		isPortalEnabled = &v
	}

	pbReq := &pb.UpdateServiceLevelRequest{
		CarrierId:       req.CarrierID,
		Id:              req.ServiceLevelID,
		Name:            req.Name,
		Code:            req.Code,
		IsPortalEnabled: isPortalEnabled,
		IsDefault:       req.IsDefault,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, serviceLevelSvcTracer, "service.service_levels.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateServiceLevelResponse, error) {
			return m.coreClient.UpdateServiceLevel(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ServiceLevelPresenter(resp.ServiceLevel)
	return &result, nil
}

func (m *serviceLevelSvcImpl) DeleteServiceLevel(ctx context.Context, req *DeleteServiceLevelRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteServiceLevelRequest{
		CarrierId: req.CarrierID,
		Id:        req.ServiceLevelID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, serviceLevelSvcTracer, "service.service_levels.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteServiceLevel(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
