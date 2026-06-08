package edidclocationep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type EDIDCLocationSvc interface {
	ListDCLocations(ctx context.Context, req *ListDCLocationsRequest) (*apiresource.List[apiresource.DCLocation], *apierror.APIError)
	GetDCLocation(ctx context.Context, req *RetrieveDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError)
	CreateDCLocation(ctx context.Context, req *CreateDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError)
	UpdateDCLocation(ctx context.Context, req *UpdateDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError)
	DeleteDCLocation(ctx context.Context, req *DeleteDCLocationRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type EDIDCLocationSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type ediDCLocationSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var ediDCLocationSvcTracer = tracing.GetTracer("api-gateway.endpoints.edi-dc-locations.service")

func (c *EDIDCLocationSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("edi dc location endpoint service: core client is required")
	}
	return nil
}

func NewEDIDCLocationSvc(config *EDIDCLocationSvcConfig) EDIDCLocationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &ediDCLocationSvcImpl{coreClient: config.CoreClient}
}

func (m *ediDCLocationSvcImpl) ListDCLocations(ctx context.Context, req *ListDCLocationsRequest) (*apiresource.List[apiresource.DCLocation], *apierror.APIError) {
	pbReq := &pb.ListDCLocationsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, ediDCLocationSvcTracer, "service.edi-dc-locations.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListDCLocationsResponse, error) {
			return m.coreClient.ListDCLocations(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	ids := make([]string, len(resp.DcLocations))
	for i, d := range resp.DcLocations {
		ids[i] = d.Id
	}
	loaded, apiErr := resourceloaders.LoadDCLocations(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.DCLocation, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.DCLocation)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *ediDCLocationSvcImpl) GetDCLocation(ctx context.Context, req *RetrieveDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError) {
	return loadDCLocationByID(ctx, req.DCLocationID)
}

func (m *ediDCLocationSvcImpl) CreateDCLocation(ctx context.Context, req *CreateDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError) {
	pbReq := &pb.CreateDCLocationRequest{
		CustomerId: req.CustomerID,
		Location:   req.Location,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, ediDCLocationSvcTracer, "service.edi-dc-locations.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateDCLocationResponse, error) {
			return m.coreClient.CreateDCLocation(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadDCLocationByID(ctx, resp.DcLocation.Id)
}

func (m *ediDCLocationSvcImpl) UpdateDCLocation(ctx context.Context, req *UpdateDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError) {
	pbReq := &pb.UpdateDCLocationRequest{
		Id:         req.DCLocationID,
		CustomerId: req.CustomerID.Ptr(),
		Location:   req.Location.Ptr(),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, ediDCLocationSvcTracer, "service.edi-dc-locations.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateDCLocationResponse, error) {
			return m.coreClient.UpdateDCLocation(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadDCLocationByID(ctx, resp.DcLocation.Id)
}

func (m *ediDCLocationSvcImpl) DeleteDCLocation(ctx context.Context, req *DeleteDCLocationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteDCLocationRequest{Id: req.DCLocationID}
	_, apiErr := grpcutil.CallRPC(ctx, ediDCLocationSvcTracer, "service.edi-dc-locations.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteDCLocation(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

func loadDCLocationByID(ctx context.Context, id string) (*apiresource.DCLocation, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadDCLocations(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("DC location not found.")
	}
	return v.(*apiresource.DCLocation), nil
}
