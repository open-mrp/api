package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fulfillmentGRPCHandler struct {
	pb.UnimplementedCoreFulfillmentServiceServer

	machineSvc       domain.MachineSvc
	machineStatusSvc domain.MachineStatusSvc
}

func machineToProto(m *domain.Machine) *pb.MachineInfo {
	if m == nil {
		return nil
	}

	info := &pb.MachineInfo{
		Id:             m.ID,
		Name:           m.Name,
		SerialNumber:   m.SerialNumber,
		Notes:          m.Notes,
		DepartmentId:   m.DepartmentID,
		DepartmentName: m.DepartmentName,
		CreatedAt:      timestamppb.New(m.CreatedAt),
		UpdatedAt:      timestamppb.New(m.UpdatedAt),
	}
	if m.DepartmentCreatedAt != nil {
		info.DepartmentCreatedAt = timestamppb.New(*m.DepartmentCreatedAt)
	}
	if m.DepartmentUpdatedAt != nil {
		info.DepartmentUpdatedAt = timestamppb.New(*m.DepartmentUpdatedAt)
	}
	return info
}

func (h *fulfillmentGRPCHandler) ListMachines(ctx context.Context, req *pb.ListMachinesRequest) (*pb.ListMachinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListMachinesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.machineSvc.ListMachines(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	machines := make([]*pb.MachineInfo, len(result.Machines))
	for i, m := range result.Machines {
		machines[i] = machineToProto(m)
	}

	return &pb.ListMachinesResponse{
		Machines: machines,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *fulfillmentGRPCHandler) GetMachine(ctx context.Context, req *pb.GetMachineRequest) (*pb.GetMachineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	machine, apiErr := h.machineSvc.GetMachine(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetMachineResponse{
		Machine: machineToProto(machine),
	}, nil
}

func (h *fulfillmentGRPCHandler) CreateMachine(ctx context.Context, req *pb.CreateMachineRequest) (*pb.CreateMachineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateMachineParams{
		Name:         req.Name,
		SerialNumber: req.SerialNumber,
		Notes:        req.Notes,
		DepartmentID: req.DepartmentId,
	}

	machine, apiErr := h.machineSvc.CreateMachine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateMachineResponse{
		Machine: machineToProto(machine),
	}, nil
}

func (h *fulfillmentGRPCHandler) UpdateMachine(ctx context.Context, req *pb.UpdateMachineRequest) (*pb.UpdateMachineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateMachineParams{
		MachineID:    req.Id,
		Name:         req.Name,
		SerialNumber: req.SerialNumber,
		Notes:        req.Notes,
	}

	machine, apiErr := h.machineSvc.UpdateMachine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateMachineResponse{
		Machine: machineToProto(machine),
	}, nil
}

func (h *fulfillmentGRPCHandler) DeleteMachine(ctx context.Context, req *pb.DeleteMachineRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.machineSvc.DeleteMachine(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *fulfillmentGRPCHandler) BatchGetMachinesByIDs(ctx context.Context, req *pb.BatchGetMachinesByIDsRequest) (*pb.BatchGetMachinesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	machines, apiErr := h.machineSvc.BatchGetMachinesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	protos := make([]*pb.MachineInfo, len(machines))
	for i, m := range machines {
		protos[i] = machineToProto(m)
	}

	return &pb.BatchGetMachinesByIDsResponse{
		Machines: protos,
	}, nil
}

func machineCampaignToProto(c *domain.MachineCampaign) *pb.MachineCampaignInfo {
	if c == nil {
		return nil
	}
	info := &pb.MachineCampaignInfo{
		ProductionScheduleLineId: c.ProductionScheduleLineID,
		ItemId:                   c.ItemID,
		Sku:                      c.SKU,
		WeekStartDate:            timestamppb.New(c.WeekStartDate),
		WeekIndex:                c.WeekIndex,
		PlannedQuantity:          c.PlannedQuantity,
		ScannedQuantity:          c.ScannedQuantity,
		RemainingQuantity:        c.RemainingQuantity,
		ReleasedBatchCount:       c.ReleasedBatchCount,
		ScannedBatchCount:        c.ScannedBatchCount,
		PlannedRunHours:          c.PlannedRunHours,
		StatusCode:               c.StatusCode,
		ProductionRunId:          c.ProductionRunID,
	}
	if c.Unit != "" {
		unit := c.Unit
		info.Unit = &unit
	}
	return info
}

func (h *fulfillmentGRPCHandler) ListMachineStatus(ctx context.Context, req *pb.ListMachineStatusRequest) (*pb.ListMachineStatusResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListMachineStatusParams{DepartmentIDs: req.DepartmentIds}
	if req.AsOf != nil {
		params.AsOf = req.AsOf.AsTime()
	}

	result, apiErr := h.machineStatusSvc.ListMachineStatus(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	machines := make([]*pb.MachineStatusInfo, 0, len(result.Machines))
	for i := range result.Machines {
		m := result.Machines[i]
		info := &pb.MachineStatusInfo{
			MachineId:           m.MachineID,
			MachineName:         m.MachineName,
			DepartmentId:        m.DepartmentID,
			DepartmentName:      m.DepartmentName,
			Status:              string(m.Status),
			Current:             machineCampaignToProto(m.Current),
			Next:                machineCampaignToProto(m.Next),
			WeekPlannedQuantity: m.WeekPlannedQuantity,
			WeekScannedQuantity: m.WeekScannedQuantity,
			WeekPlannedRunHours: m.WeekPlannedRunHours,
		}
		if m.Unit != "" {
			unit := m.Unit
			info.Unit = &unit
		}
		if m.Downtime != nil {
			info.Downtime = &pb.MachineDowntimeSummaryInfo{
				EventId:    m.Downtime.EventID,
				Reason:     m.Downtime.Reason,
				ReasonName: m.Downtime.ReasonName,
				OeeBucket:  m.Downtime.OEEBucket,
				StartedAt:  timestamppb.New(m.Downtime.StartedAt),
				Note:       m.Downtime.Note,
			}
		}
		machines = append(machines, info)
	}

	resp := &pb.ListMachineStatusResponse{
		WeekStartDate: timestamppb.New(result.WeekStartDate),
		Machines:      machines,
	}
	if result.ProductionScheduleID != "" {
		scheduleID := result.ProductionScheduleID
		resp.ProductionScheduleId = &scheduleID
	}
	return resp, nil
}
