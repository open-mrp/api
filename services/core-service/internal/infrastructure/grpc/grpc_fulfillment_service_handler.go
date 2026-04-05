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

	machineSvc domain.MachineSvc
}

func machineToProto(m *domain.Machine) *pb.MachineInfo {
	if m == nil {
		return nil
	}

	return &pb.MachineInfo{
		Id:             m.ID,
		Name:           m.Name,
		SerialNumber:   m.SerialNumber,
		Notes:          m.Notes,
		DepartmentId:   m.DepartmentID,
		DepartmentName: m.DepartmentName,
		CreatedAt:      timestamppb.New(m.CreatedAt),
		UpdatedAt:      timestamppb.New(m.UpdatedAt),
	}
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
