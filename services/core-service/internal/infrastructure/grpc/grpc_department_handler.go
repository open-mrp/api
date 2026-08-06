package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func departmentToProto(d *domain.Department) *pb.DepartmentInfo {
	info := &pb.DepartmentInfo{
		Id:        d.ID,
		Name:      d.Name,
		Notes:     d.Notes,
		CreatedAt: timestamppb.New(d.CreatedAt),
		UpdatedAt: timestamppb.New(d.UpdatedAt),
	}

	if d.LocationID != nil {
		info.LocationId = d.LocationID
	}
	if d.LocationName != nil {
		info.LocationName = d.LocationName
	}
	if d.LocationTypeCode != nil {
		info.LocationTypeCode = d.LocationTypeCode
	}
	if d.LaborRate != nil {
		info.LaborRate = &pb.DepartmentRateInfo{
			Id:                          d.LaborRate.ID,
			Value:                       d.LaborRate.Value,
			NumeratorUnitId:             d.LaborRate.NumeratorUnit.ID,
			NumeratorUnitAbbreviation:   d.LaborRate.NumeratorUnit.Abbreviation,
			NumeratorUnitType:           d.LaborRate.NumeratorUnit.Type,
			DenominatorUnitId:           d.LaborRate.DenominatorUnit.ID,
			DenominatorUnitAbbreviation: d.LaborRate.DenominatorUnit.Abbreviation,
			DenominatorUnitType:         d.LaborRate.DenominatorUnit.Type,
		}
	}

	stations := make([]*pb.LightScanningStationInfo, len(d.ScanningStations))
	for i, s := range d.ScanningStations {
		stations[i] = &pb.LightScanningStationInfo{
			Id:                  s.ID,
			Name:                s.Name,
			Type:                s.Type,
			OperatorRequirement: s.OperatorRequirement,
			CreatedAt:           timestamppb.New(s.CreatedAt),
			UpdatedAt:           timestamppb.New(s.UpdatedAt),
		}
	}
	info.ScanningStations = stations

	machines := make([]*pb.LightMachineInfo, len(d.Machines))
	for i, m := range d.Machines {
		machines[i] = &pb.LightMachineInfo{
			Id:           m.ID,
			Name:         m.Name,
			SerialNumber: m.SerialNumber,
			CreatedAt:    timestamppb.New(m.CreatedAt),
			UpdatedAt:    timestamppb.New(m.UpdatedAt),
		}
	}
	info.Machines = machines

	return info
}

func (h *gRPCHandler) ExportDepartments(ctx context.Context, req *pb.ExportDepartmentsRequest) (*pb.ExportDepartmentsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	job, apiErr := h.departmentSvc.ExportDepartments(ctx, domain.ExportDepartmentsParams{Query: req.Query})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ExportDepartmentsResponse{Job: jobToProto(job)}, nil
}

func departmentRateParamsFromProto(in *pb.DepartmentRateInput) *domain.CreateRateParams {
	if in == nil {
		return nil
	}
	return &domain.CreateRateParams{
		Value:             in.Value,
		NumeratorUnitID:   in.NumeratorUnitId,
		DenominatorUnitID: in.DenominatorUnitId,
	}
}

func (h *gRPCHandler) ListDepartments(ctx context.Context, req *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListDepartmentsParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.departmentSvc.ListDepartments(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbDepts := make([]*pb.DepartmentInfo, len(result.Departments))
	for i, d := range result.Departments {
		pbDepts[i] = departmentToProto(d)
	}

	return &pb.ListDepartmentsResponse{
		Departments: pbDepts,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetDepartment(ctx context.Context, req *pb.GetDepartmentRequest) (*pb.GetDepartmentResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	dept, apiErr := h.departmentSvc.GetDepartment(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetDepartmentResponse{
		Department: departmentToProto(dept),
	}, nil
}

func (h *gRPCHandler) CreateDepartment(ctx context.Context, req *pb.CreateDepartmentRequest) (*pb.CreateDepartmentResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateDepartmentParams{
		Name:               req.Name,
		Notes:              req.Notes,
		LocationID:         req.LocationId,
		LaborRate:          departmentRateParamsFromProto(req.LaborRate),
		ScanningStationIDs: req.ScanningStationIds,
		MachineIDs:         req.MachineIds,
	}

	dept, apiErr := h.departmentSvc.CreateDepartment(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateDepartmentResponse{
		Department: departmentToProto(dept),
	}, nil
}

func (h *gRPCHandler) BulkUpsertDepartments(ctx context.Context, req *pb.BulkUpsertDepartmentsRequest) (*pb.BulkUpsertDepartmentsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	departments := make([]domain.UpsertDepartmentParams, len(req.Departments))
	for i, d := range req.Departments {
		departments[i] = domain.UpsertDepartmentParams{
			Name:     d.Name,
			Notes:    d.Notes,
			Location: objectIdentifierPtrFromProto(d.Location),
		}
	}

	job, apiErr := h.departmentSvc.BulkUpsertDepartments(ctx, domain.BulkUpsertDepartmentsParams{Departments: departments})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.BulkUpsertDepartmentsResponse{Job: jobToProto(job)}, nil
}

func (h *gRPCHandler) UpdateDepartment(ctx context.Context, req *pb.UpdateDepartmentRequest) (*pb.UpdateDepartmentResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateDepartmentParams{
		DepartmentID:       req.Id,
		Name:               req.Name,
		Notes:              req.Notes,
		LocationID:         req.LocationId,
		LaborRate:          departmentRateParamsFromProto(req.LaborRate),
		ScanningStationIDs: req.ScanningStationIds,
		MachineIDs:         req.MachineIds,
	}

	dept, apiErr := h.departmentSvc.UpdateDepartment(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateDepartmentResponse{
		Department: departmentToProto(dept),
	}, nil
}

func (h *gRPCHandler) DeleteDepartment(ctx context.Context, req *pb.DeleteDepartmentRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.departmentSvc.DeleteDepartment(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) BatchGetDepartmentsByIDs(ctx context.Context, req *pb.BatchGetDepartmentsByIDsRequest) (*pb.BatchGetDepartmentsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	departments, apiErr := h.departmentSvc.BatchGetDepartmentsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbDepts := make([]*pb.DepartmentInfo, len(departments))
	for i, d := range departments {
		pbDepts[i] = departmentToProto(d)
	}

	return &pb.BatchGetDepartmentsByIDsResponse{
		Departments: pbDepts,
	}, nil
}
