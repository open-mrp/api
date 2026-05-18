package locationep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type LocationSvc interface {
	ListLocations(ctx context.Context, req *ListLocationsRequest) (*apiresource.List[apiresource.Location], *apierror.APIError)
	GetLocation(ctx context.Context, req *RetrieveLocationRequest) (*apiresource.Location, *apierror.APIError)
	CreateLocation(ctx context.Context, req *CreateLocationRequest) (*apiresource.Location, *apierror.APIError)
	UpdateLocation(ctx context.Context, req *UpdateLocationRequest) (*apiresource.Location, *apierror.APIError)
	DeleteLocation(ctx context.Context, req *DeleteLocationRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListLocationTypes(ctx context.Context, req *ListLocationTypesRequest) (*apiresource.List[apiresource.LocationType], *apierror.APIError)
	GetLocationType(ctx context.Context, req *RetrieveLocationTypeRequest) (*apiresource.LocationType, *apierror.APIError)
}

type LocationSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type locationSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var locationSvcTracer = tracing.GetTracer("api-gateway.endpoints.locations.service")

func (c *LocationSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("location endpoint service: core client is required")
	}
	return nil
}

func NewLocationSvc(config *LocationSvcConfig) LocationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &locationSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *locationSvcImpl) ListLocations(ctx context.Context, req *ListLocationsRequest) (*apiresource.List[apiresource.Location], *apierror.APIError) {
	pbReq := &pb.ListLocationsRequest{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListLocationsResponse, error) {
			return m.coreClient.ListLocations(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return LocationListPresenter(ctx, resp), nil
}

func (m *locationSvcImpl) GetLocation(ctx context.Context, req *RetrieveLocationRequest) (*apiresource.Location, *apierror.APIError) {
	pbReq := &pb.GetLocationRequest{
		Id:       req.LocationID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetLocationResponse, error) {
			return m.coreClient.GetLocation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := LocationPresenter(resp.Location)
	return &result, nil
}

func (m *locationSvcImpl) CreateLocation(ctx context.Context, req *CreateLocationRequest) (*apiresource.Location, *apierror.APIError) {
	pbReq := &pb.CreateLocationRequest{
		Name:     req.Name,
		TypeCode: string(req.TypeCode),
		ParentId: req.ParentID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}
	if req.ChildIDs != nil {
		pbReq.ChildIds = *req.ChildIDs
	}

	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateLocationResponse, error) {
			return m.coreClient.CreateLocation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := LocationPresenter(resp.Location)
	return &result, nil
}

func (m *locationSvcImpl) UpdateLocation(ctx context.Context, req *UpdateLocationRequest) (*apiresource.Location, *apierror.APIError) {
	pbReq := &pb.UpdateLocationRequest{
		Id:       req.LocationID,
		Name:     req.Name,
		TypeCode: req.TypeCode.StringPtr(),
		ParentId: req.ParentID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}
	if req.ChildIDs.IsSet() {
		pbReq.UpdateChildren = true
		if !req.ChildIDs.IsNull() {
			pbReq.ChildIds = *req.ChildIDs.Value()
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateLocationResponse, error) {
			return m.coreClient.UpdateLocation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := LocationPresenter(resp.Location)
	return &result, nil
}

func (m *locationSvcImpl) DeleteLocation(ctx context.Context, req *DeleteLocationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteLocationRequest{
		Id: req.LocationID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteLocation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *locationSvcImpl) ListLocationTypes(ctx context.Context, req *ListLocationTypesRequest) (*apiresource.List[apiresource.LocationType], *apierror.APIError) {
	pbReq := &pb.ListLocationTypesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.list_types", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListLocationTypesResponse, error) {
			return m.coreClient.ListLocationTypes(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return LocationTypeListPresenter(ctx, resp), nil
}

func (m *locationSvcImpl) GetLocationType(ctx context.Context, req *RetrieveLocationTypeRequest) (*apiresource.LocationType, *apierror.APIError) {
	pbReq := &pb.GetLocationTypeRequest{
		Identifier: req.Identifier,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.get_type", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetLocationTypeResponse, error) {
			return m.coreClient.GetLocationType(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := LocationTypePresenter(resp.LocationType)
	return &result, nil
}
