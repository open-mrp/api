package workcenterep

import (
	"context"
	"fmt"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// WorkCenterSvc serves work centers. They are stored as departments, so the core RPCs keep that name.
type WorkCenterSvc interface {
	ListWorkCenters(ctx context.Context, req *ListWorkCentersRequest) (*apiresource.List[apiresource.WorkCenter], *apierror.APIError)
	ExportWorkCenters(ctx context.Context, req *ExportWorkCentersRequest) (*apiresource.AsyncJob, *apierror.APIError)
	GetWorkCenter(ctx context.Context, req *RetrieveWorkCenterRequest) (*apiresource.WorkCenter, *apierror.APIError)
	CreateWorkCenter(ctx context.Context, req *CreateWorkCenterRequest) (*apiresource.WorkCenter, *apierror.APIError)
	UpdateWorkCenter(ctx context.Context, req *UpdateWorkCenterRequest) (*apiresource.WorkCenter, *apierror.APIError)
	BulkUpsertWorkCenters(ctx context.Context, req *BulkUpsertWorkCentersRequest) (*apiresource.AsyncJob, *apierror.APIError)
	DeleteWorkCenter(ctx context.Context, req *DeleteWorkCenterRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type WorkCenterSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type workCenterSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var workCenterSvcTracer = tracing.GetTracer("api-gateway.endpoints.work_centers.service")

func (c *WorkCenterSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("work center endpoint service: core client is required")
	}
	return nil
}

func NewWorkCenterSvc(config *WorkCenterSvcConfig) WorkCenterSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &workCenterSvcImpl{coreClient: config.CoreClient}
}

func (m *workCenterSvcImpl) ExportWorkCenters(ctx context.Context, req *ExportWorkCentersRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, workCenterSvcTracer, "service.work_centers.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportDepartmentsResponse, error) {
			return m.coreClient.ExportDepartments(ctx, &pb.ExportDepartmentsRequest{Query: req.Query}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *workCenterSvcImpl) ListWorkCenters(ctx context.Context, req *ListWorkCentersRequest) (*apiresource.List[apiresource.WorkCenter], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, workCenterSvcTracer, "service.work_centers.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListDepartmentsResponse, error) {
			return m.coreClient.ListDepartments(ctx, &pb.ListDepartmentsRequest{Cursor: req.Cursor, Limit: req.Limit, Query: req.Query}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Departments))
	for i, d := range resp.Departments {
		ids[i] = d.Id
	}
	loaded, apiErr := resourceloaders.LoadWorkCenters(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	workCenters := make([]apiresource.WorkCenter, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			workCenters = append(workCenters, *(v.(*apiresource.WorkCenter)))
		}
	}
	return apiresource.NewList(workCenters, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *workCenterSvcImpl) GetWorkCenter(ctx context.Context, req *RetrieveWorkCenterRequest) (*apiresource.WorkCenter, *apierror.APIError) {
	return loadWorkCenterByID(ctx, req.WorkCenterID)
}

func (m *workCenterSvcImpl) CreateWorkCenter(ctx context.Context, req *CreateWorkCenterRequest) (*apiresource.WorkCenter, *apierror.APIError) {
	pbReq := &pb.CreateDepartmentRequest{
		Name:               req.Name,
		LocationId:         req.LocationID.Ptr(),
		ScanningStationIds: req.ScanningStationIDs,
		MachineIds:         req.MachineIDs,
		LaborRate:          apirequest.OptionalUnitPriceToProto(req.LaborRate),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, workCenterSvcTracer, "service.work_centers.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateDepartmentResponse, error) {
			return m.coreClient.CreateDepartment(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadWorkCenterByID(ctx, resp.Department.Id)
}

func (m *workCenterSvcImpl) BulkUpsertWorkCenters(ctx context.Context, req *BulkUpsertWorkCentersRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	rows := make([]*pb.UpsertDepartmentInput, len(req.WorkCenters))
	for i, wc := range req.WorkCenters {
		rows[i] = &pb.UpsertDepartmentInput{
			Name:     wc.Name,
			Location: apirequest.OptionalObjectIdentifierToProto(wc.Location),
		}
	}
	resp, apiErr := grpcutil.CallRPC(ctx, workCenterSvcTracer, "service.work_centers.bulk_upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkUpsertDepartmentsResponse, error) {
			return m.coreClient.BulkUpsertDepartments(ctx, &pb.BulkUpsertDepartmentsRequest{Departments: rows}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *workCenterSvcImpl) UpdateWorkCenter(ctx context.Context, req *UpdateWorkCenterRequest) (*apiresource.WorkCenter, *apierror.APIError) {
	pbReq := &pb.UpdateDepartmentRequest{
		Id:                 req.WorkCenterID,
		Name:               req.Name.Ptr(),
		LocationId:         req.LocationID.Ptr(),
		ScanningStationIds: req.ScanningStationIDs,
		MachineIds:         req.MachineIDs,
		LaborRate:          apirequest.OptionalUnitPriceToProto(req.LaborRate),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, workCenterSvcTracer, "service.work_centers.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateDepartmentResponse, error) {
			return m.coreClient.UpdateDepartment(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadWorkCenterByID(ctx, resp.Department.Id)
}

func (m *workCenterSvcImpl) DeleteWorkCenter(ctx context.Context, req *DeleteWorkCenterRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, workCenterSvcTracer, "service.work_centers.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteDepartment(ctx, &pb.DeleteDepartmentRequest{Id: req.WorkCenterID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

func loadWorkCenterByID(ctx context.Context, id string) (*apiresource.WorkCenter, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadWorkCenters(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewNotFoundError("Work center not found.")
	}
	return v.(*apiresource.WorkCenter), nil
}
