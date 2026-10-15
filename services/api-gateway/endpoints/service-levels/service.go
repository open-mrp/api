package servicelevelep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ServiceLevelSvc interface {
	ListServiceLevels(ctx context.Context, req *ListServiceLevelsRequest) (*apiresource.List[apiresource.ServiceLevel], *apierror.APIError)
	GetServiceLevel(ctx context.Context, req *RetrieveServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError)
	CreateServiceLevel(ctx context.Context, req *CreateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError)
	UpdateServiceLevel(ctx context.Context, req *UpdateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError)
	DeleteServiceLevel(ctx context.Context, req *DeleteServiceLevelRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ServiceLevelSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
	return &serviceLevelSvcImpl{coreClient: config.CoreClient}
}

func (m *serviceLevelSvcImpl) ListServiceLevels(ctx context.Context, req *ListServiceLevelsRequest) (*apiresource.List[apiresource.ServiceLevel], *apierror.APIError) {
	pbReq := &pb.ListServiceLevelsRequest{
		CarrierId: req.CarrierID,
		Cursor:    req.Cursor,
		Limit:     req.Limit,
		Query:     req.Query,
		Statuses:  archiveStatusStrings(req.Statuses),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, serviceLevelSvcTracer, "service.service_levels.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListServiceLevelsResponse, error) {
			return m.coreClient.ListServiceLevels(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	ids := make([]string, len(resp.ServiceLevels))
	for i, sl := range resp.ServiceLevels {
		ids[i] = sl.Id
	}
	loaded, apiErr := resourceloaders.LoadServiceLevels(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.ServiceLevel, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.ServiceLevel)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *serviceLevelSvcImpl) GetServiceLevel(ctx context.Context, req *RetrieveServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, serviceLevelSvcTracer, "service.service_levels.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetServiceLevelResponse, error) {
			return m.coreClient.GetServiceLevel(ctx, &pb.GetServiceLevelRequest{
				CarrierId: req.CarrierID,
				Id:        req.ServiceLevelID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadServiceLevelByID(ctx, resp.ServiceLevel.Id)
}

func (m *serviceLevelSvcImpl) CreateServiceLevel(ctx context.Context, req *CreateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
	isPortalEnabled := true
	if v, ok := req.PortalVisibility.Value(); ok {
		isPortalEnabled = v == constants.CustomerPortalVisibilityVisible
	}
	makeDefault, _ := req.IsDefault.Value()
	pbReq := &pb.CreateServiceLevelRequest{
		CarrierId:            req.CarrierID,
		Name:                 req.Name,
		Code:                 req.Code,
		IsPortalEnabled:      isPortalEnabled,
		MakeDefault:          makeDefault,
		EstimatedTransitDays: req.EstimatedTransitDays.Ptr(),
		Status:               archiveStatusPtr(req.Status),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, serviceLevelSvcTracer, "service.service_levels.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateServiceLevelResponse, error) {
			return m.coreClient.CreateServiceLevel(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadServiceLevelByID(ctx, resp.ServiceLevel.Id)
}

func (m *serviceLevelSvcImpl) UpdateServiceLevel(ctx context.Context, req *UpdateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
	var isPortalEnabled *bool
	if v, ok := req.PortalVisibility.Value(); ok {
		enabled := v == constants.CustomerPortalVisibilityVisible
		isPortalEnabled = &enabled
	}
	pbReq := &pb.UpdateServiceLevelRequest{
		CarrierId:            req.CarrierID,
		Id:                   req.ServiceLevelID,
		Name:                 req.Name.Ptr(),
		Code:                 req.Code.Ptr(),
		IsPortalEnabled:      isPortalEnabled,
		MakeDefault:          req.IsDefault.Ptr(),
		EstimatedTransitDays: fieldpb.Int32ClearableToProto(req.EstimatedTransitDays),
		Status:               archiveStatusPtr(req.Status),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, serviceLevelSvcTracer, "service.service_levels.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateServiceLevelResponse, error) {
			return m.coreClient.UpdateServiceLevel(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadServiceLevelByID(ctx, resp.ServiceLevel.Id)
}

func (m *serviceLevelSvcImpl) DeleteServiceLevel(ctx context.Context, req *DeleteServiceLevelRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, serviceLevelSvcTracer, "service.service_levels.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteServiceLevel(ctx, &pb.DeleteServiceLevelRequest{
				CarrierId: req.CarrierID,
				Id:        req.ServiceLevelID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

// loadServiceLevelByID wraps the single-ID load used after mutations + on
// retrieve. Same pattern as carriers/payment-terms.
func loadServiceLevelByID(ctx context.Context, id string) (*apiresource.ServiceLevel, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadServiceLevels(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewNotFoundError("Service level not found.")
	}
	return v.(*apiresource.ServiceLevel), nil
}

// archiveStatusPtr is a request's optional status as core takes it.
func archiveStatusPtr(status field.Optional[constants.ArchiveStatus]) *string {
	v, ok := status.Value()
	if !ok {
		return nil
	}
	str := string(v)
	return &str
}

// archiveStatusStrings is a statuses filter as core takes it.
func archiveStatusStrings(statuses []constants.ArchiveStatus) []string {
	out := make([]string, len(statuses))
	for i, st := range statuses {
		out[i] = string(st)
	}
	return out
}
