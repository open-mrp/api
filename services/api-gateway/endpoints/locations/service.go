package locationep

import (
	"context"
	"fmt"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type LocationSvc interface {
	ListLocations(ctx context.Context, req *ListLocationsRequest) (*apiresource.List[apiresource.Location], *apierror.APIError)
	ExportLocations(ctx context.Context, req *ExportLocationsRequest) (*apiresource.Job, *apierror.APIError)
	GetLocation(ctx context.Context, req *RetrieveLocationRequest) (*apiresource.Location, *apierror.APIError)
	CreateLocation(ctx context.Context, req *CreateLocationRequest) (*apiresource.Location, *apierror.APIError)
	UpdateLocation(ctx context.Context, req *UpdateLocationRequest) (*apiresource.Location, *apierror.APIError)
	DeleteLocation(ctx context.Context, req *DeleteLocationRequest) (*apiresource.EmptyResource, *apierror.APIError)
	BulkUpsertLocations(ctx context.Context, req *BulkUpsertLocationsRequest) (*apiresource.Job, *apierror.APIError)
	ListLocationTypes(ctx context.Context, req *ListLocationTypesRequest) (*apiresource.List[apiresource.LocationType], *apierror.APIError)
	GetLocationType(ctx context.Context, req *RetrieveLocationTypeRequest) (*apiresource.LocationType, *apierror.APIError)
}

type LocationSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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

func (m *locationSvcImpl) ExportLocations(ctx context.Context, req *ExportLocationsRequest) (*apiresource.Job, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportLocationsResponse, error) {
			return m.coreClient.ExportLocations(ctx, &pb.ExportLocationsRequest{Query: req.Query}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *locationSvcImpl) ListLocations(ctx context.Context, req *ListLocationsRequest) (*apiresource.List[apiresource.Location], *apierror.APIError) {
	pbReq := &pb.ListLocationsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListLocationsResponse, error) {
			return m.coreClient.ListLocations(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Locations))
	for i, loc := range resp.Locations {
		ids[i] = loc.Id
	}
	loaded, apiErr := resourceloaders.LoadLocations(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	locations := make([]apiresource.Location, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			locations = append(locations, *(v.(*apiresource.Location)))
		}
	}
	return apiresource.NewList(locations, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *locationSvcImpl) GetLocation(ctx context.Context, req *RetrieveLocationRequest) (*apiresource.Location, *apierror.APIError) {
	return loadLocationByID(ctx, req.LocationID)
}

func (m *locationSvcImpl) CreateLocation(ctx context.Context, req *CreateLocationRequest) (*apiresource.Location, *apierror.APIError) {
	pbReq := &pb.CreateLocationRequest{
		Name:     req.Name,
		TypeCode: string(req.TypeCode),
		ParentId: req.ParentID.Ptr(),
	}
	if childIDs := req.ChildIDs.Ptr(); childIDs != nil {
		pbReq.ChildIds = *childIDs
	}

	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateLocationResponse, error) {
			return m.coreClient.CreateLocation(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadLocationByID(ctx, resp.Location.Id)
}

func (m *locationSvcImpl) UpdateLocation(ctx context.Context, req *UpdateLocationRequest) (*apiresource.Location, *apierror.APIError) {
	pbReq := &pb.UpdateLocationRequest{
		Id:       req.LocationID,
		Name:     req.Name.Ptr(),
		TypeCode: req.TypeCode.Ptr().StringPtr(),
		ParentId: field.StringClearableToProto(req.ParentID),
	}
	pbReq.ChildIds = field.StringListSliceClearableToProto(req.ChildIDs)

	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateLocationResponse, error) {
			return m.coreClient.UpdateLocation(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadLocationByID(ctx, resp.Location.Id)
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

func loadLocationByID(ctx context.Context, id string) (*apiresource.Location, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadLocations(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Location not found.")
	}
	return v.(*apiresource.Location), nil
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

func (m *locationSvcImpl) BulkUpsertLocations(ctx context.Context, req *BulkUpsertLocationsRequest) (*apiresource.Job, *apierror.APIError) {
	locations := make([]*pb.BulkUpsertLocationInput, 0, len(req.Locations))
	for _, loc := range req.Locations {
		children := make([]*pb.ObjectIdentifier, 0, len(loc.Children))
		for _, c := range loc.Children {
			children = append(children, apirequest.ObjectIdentifierToProto(c))
		}
		locations = append(locations, &pb.BulkUpsertLocationInput{
			Name:     loc.Name,
			TypeCode: string(loc.TypeCode),
			Parent:   apirequest.ObjectIdentifierPtrToProto(loc.Parent),
			Children: children,
		})
	}

	resp, apiErr := grpcutil.CallRPC(ctx, locationSvcTracer, "service.locations.bulk_upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkUpsertLocationsResponse, error) {
			return m.coreClient.BulkUpsertLocations(ctx, &pb.BulkUpsertLocationsRequest{Locations: locations}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
}
