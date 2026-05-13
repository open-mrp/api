package syspropertyep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type SysPropertySvc interface {
	ListSysProperties(ctx context.Context, req *ListSysPropertiesRequest) (*apiresource.List[apiresource.SysProperty], *apierror.APIError)
	GetSysProperty(ctx context.Context, req *RetrieveSysPropertyRequest) (*apiresource.SysProperty, *apierror.APIError)
	UpdateSysProperty(ctx context.Context, req *UpdateSysPropertyRequest) (*apiresource.SysProperty, *apierror.APIError)
	GetLatestSysPropertyValue(ctx context.Context, req *GetLatestSysPropertyValueRequest) (*apiresource.SysPropertyValue, *apierror.APIError)
}

type SysPropertySvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type sysPropertySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var sysPropertySvcTracer = tracing.GetTracer("api-gateway.endpoints.sys_properties.service")

func (c *SysPropertySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sys property endpoint service: core client is required")
	}
	return nil
}

func NewSysPropertySvc(config *SysPropertySvcConfig) SysPropertySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &sysPropertySvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *sysPropertySvcImpl) ListSysProperties(ctx context.Context, req *ListSysPropertiesRequest) (*apiresource.List[apiresource.SysProperty], *apierror.APIError) {
	pbReq := &pb.ListSysPropertiesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, sysPropertySvcTracer, "service.sys_properties.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSysPropertiesResponse, error) {
			return m.coreClient.ListSysProperties(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return SysPropertyListPresenter(resp), nil
}

func (m *sysPropertySvcImpl) GetSysProperty(ctx context.Context, req *RetrieveSysPropertyRequest) (*apiresource.SysProperty, *apierror.APIError) {
	pbReq := &pb.GetSysPropertyRequest{
		Id: req.SysPropertyID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, sysPropertySvcTracer, "service.sys_properties.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSysPropertyResponse, error) {
			return m.coreClient.GetSysProperty(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := SysPropertyPresenter(resp.SysProperty)
	return &result, nil
}

func (m *sysPropertySvcImpl) UpdateSysProperty(ctx context.Context, req *UpdateSysPropertyRequest) (*apiresource.SysProperty, *apierror.APIError) {
	pbReq := &pb.UpdateSysPropertyRequest{
		Id:    req.SysPropertyID,
		Value: req.Value,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, sysPropertySvcTracer, "service.sys_properties.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSysPropertyResponse, error) {
			return m.coreClient.UpdateSysProperty(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := SysPropertyPresenter(resp.SysProperty)
	return &result, nil
}

func (m *sysPropertySvcImpl) GetLatestSysPropertyValue(ctx context.Context, req *GetLatestSysPropertyValueRequest) (*apiresource.SysPropertyValue, *apierror.APIError) {
	pbReq := &pb.GetLatestSysPropertyValueRequest{
		TypeCode: req.TypeCode,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, sysPropertySvcTracer, "service.sys_properties.get_latest_value", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetLatestSysPropertyValueResponse, error) {
			return m.coreClient.GetLatestSysPropertyValue(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.SysPropertyValue{
		Object: "sys_property_value",
		Value:  fmt.Sprintf("%d", resp.Value),
	}, nil
}
