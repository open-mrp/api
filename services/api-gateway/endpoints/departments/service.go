package departmentep

import (
	"context"
	"fmt"

	jobep "github.com/augno/api/services/api-gateway/endpoints/jobs"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DepartmentSvc interface {
	ListDepartments(ctx context.Context, req *ListDepartmentsRequest) (*apiresource.List[apiresource.Department], *apierror.APIError)
	ExportDepartments(ctx context.Context, req *ExportDepartmentsRequest) (*apiresource.Job, *apierror.APIError)
	GetDepartment(ctx context.Context, req *RetrieveDepartmentRequest) (*apiresource.Department, *apierror.APIError)
	CreateDepartment(ctx context.Context, req *CreateDepartmentRequest) (*apiresource.Department, *apierror.APIError)
	UpdateDepartment(ctx context.Context, req *UpdateDepartmentRequest) (*apiresource.Department, *apierror.APIError)
	BulkUpsertDepartments(ctx context.Context, req *BulkUpsertDepartmentsRequest) (*apiresource.Job, *apierror.APIError)
	DeleteDepartment(ctx context.Context, req *DeleteDepartmentRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type DepartmentSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type departmentSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var departmentSvcTracer = tracing.GetTracer("api-gateway.endpoints.departments.service")

func (c *DepartmentSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("department endpoint service: core client is required")
	}
	return nil
}

func NewDepartmentSvc(config *DepartmentSvcConfig) DepartmentSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &departmentSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *departmentSvcImpl) ExportDepartments(ctx context.Context, req *ExportDepartmentsRequest) (*apiresource.Job, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, departmentSvcTracer, "service.departments.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportDepartmentsResponse, error) {
			return m.coreClient.ExportDepartments(ctx, &pb.ExportDepartmentsRequest{Query: req.Query}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *departmentSvcImpl) ListDepartments(ctx context.Context, req *ListDepartmentsRequest) (*apiresource.List[apiresource.Department], *apierror.APIError) {
	pbReq := &pb.ListDepartmentsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, departmentSvcTracer, "service.departments.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListDepartmentsResponse, error) {
			return m.coreClient.ListDepartments(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Departments))
	for i, d := range resp.Departments {
		ids[i] = d.Id
	}
	loaded, apiErr := resourceloaders.LoadDepartments(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	departments := make([]apiresource.Department, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			departments = append(departments, *(v.(*apiresource.Department)))
		}
	}
	return apiresource.NewList(departments, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *departmentSvcImpl) GetDepartment(ctx context.Context, req *RetrieveDepartmentRequest) (*apiresource.Department, *apierror.APIError) {
	return loadDepartmentByID(ctx, req.DepartmentID)
}

func (m *departmentSvcImpl) CreateDepartment(ctx context.Context, req *CreateDepartmentRequest) (*apiresource.Department, *apierror.APIError) {
	pbReq := &pb.CreateDepartmentRequest{
		Name:               req.Name,
		Notes:              req.Notes.Ptr(),
		LocationId:         req.LocationID.Ptr(),
		ScanningStationIds: req.ScanningStationIDs,
		MachineIds:         req.MachineIDs,
		LaborRate:          departmentRateInputToProto(req.LaborRate),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, departmentSvcTracer, "service.departments.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateDepartmentResponse, error) {
			return m.coreClient.CreateDepartment(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadDepartmentByID(ctx, resp.Department.Id)
}

func (m *departmentSvcImpl) BulkUpsertDepartments(ctx context.Context, req *BulkUpsertDepartmentsRequest) (*apiresource.Job, *apierror.APIError) {
	pbDepartments := make([]*pb.UpsertDepartmentInput, len(req.Departments))
	for i, d := range req.Departments {
		pbDepartments[i] = &pb.UpsertDepartmentInput{
			Name:     d.Name,
			Notes:    d.Notes.Ptr(),
			Location: apirequest.OptionalObjectIdentifierToProto(d.Location),
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, departmentSvcTracer, "service.departments.bulk_upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkUpsertDepartmentsResponse, error) {
			return m.coreClient.BulkUpsertDepartments(ctx, &pb.BulkUpsertDepartmentsRequest{Departments: pbDepartments}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *departmentSvcImpl) UpdateDepartment(ctx context.Context, req *UpdateDepartmentRequest) (*apiresource.Department, *apierror.APIError) {
	pbReq := &pb.UpdateDepartmentRequest{
		Id:                 req.DepartmentID,
		Name:               req.Name.Ptr(),
		Notes:              req.Notes.Ptr(),
		LocationId:         req.LocationID.Ptr(),
		ScanningStationIds: req.ScanningStationIDs,
		MachineIds:         req.MachineIDs,
		LaborRate:          departmentRateInputToProto(req.LaborRate),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, departmentSvcTracer, "service.departments.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateDepartmentResponse, error) {
			return m.coreClient.UpdateDepartment(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadDepartmentByID(ctx, resp.Department.Id)
}

func (m *departmentSvcImpl) DeleteDepartment(ctx context.Context, req *DeleteDepartmentRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteDepartmentRequest{
		Id: req.DepartmentID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, departmentSvcTracer, "service.departments.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteDepartment(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func departmentRateInputToProto(in *DepartmentRateInput) *pb.DepartmentRateInput {
	if in == nil {
		return nil
	}
	return &pb.DepartmentRateInput{
		Value:             in.Value,
		NumeratorUnitId:   in.NumeratorUnitID,
		DenominatorUnitId: in.DenominatorUnitID,
	}
}

func loadDepartmentByID(ctx context.Context, id string) (*apiresource.Department, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadDepartments(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Department not found.")
	}
	return v.(*apiresource.Department), nil
}
