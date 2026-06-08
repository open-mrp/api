package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func scanningStationToProto(ss *domain.ScanningStation) *pb.ScanningStationInfo {
	info := &pb.ScanningStationInfo{
		Id:                  ss.ID,
		Name:                ss.Name,
		Notes:               ss.Notes,
		Type:                string(ss.Type),
		LabelSizeCode:       ss.LabelSizeCode,
		LabelTypeCode:       ss.LabelTypeCode,
		OperatorRequirement: string(ss.OperatorRequirement),
		DepartmentId:        ss.DepartmentID,
		DepartmentName:      ss.DepartmentName,
		CreatedAt:           timestamppb.New(ss.CreatedAt),
		UpdatedAt:           timestamppb.New(ss.UpdatedAt),
	}

	if ss.DepartmentCreatedAt != nil {
		info.DepartmentCreatedAt = timestamppb.New(*ss.DepartmentCreatedAt)
	}
	if ss.DepartmentUpdatedAt != nil {
		info.DepartmentUpdatedAt = timestamppb.New(*ss.DepartmentUpdatedAt)
	}

	steps := make([]*pb.LightProductionStepInfo, len(ss.ProductionSteps))
	for i, s := range ss.ProductionSteps {
		steps[i] = &pb.LightProductionStepInfo{
			Id:             s.ID,
			Name:           s.Name,
			LevelingFactor: &s.LevelingFactor,
			Allowances:     &s.Allowances,
			CreatedAt:      timestamppb.New(s.CreatedAt),
			UpdatedAt:      timestamppb.New(s.UpdatedAt),
		}
	}
	info.ProductionSteps = steps

	return info
}

func (h *gRPCHandler) BatchGetScanningStationsByIDs(ctx context.Context, req *pb.BatchGetScanningStationsByIDsRequest) (*pb.BatchGetScanningStationsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	stations, apiErr := h.scanningStationSvc.BatchGetScanningStationsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbStations := make([]*pb.ScanningStationInfo, len(stations))
	for i, ss := range stations {
		pbStations[i] = scanningStationToProto(ss)
	}

	return &pb.BatchGetScanningStationsByIDsResponse{
		ScanningStations: pbStations,
	}, nil
}

func (h *gRPCHandler) ListScanningStations(ctx context.Context, req *pb.ListScanningStationsRequest) (*pb.ListScanningStationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListScanningStationsParams{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Includes: req.Includes,
	}

	result, apiErr := h.scanningStationSvc.ListScanningStations(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbStations := make([]*pb.ScanningStationInfo, len(result.ScanningStations))
	for i, ss := range result.ScanningStations {
		pbStations[i] = scanningStationToProto(ss)
	}

	return &pb.ListScanningStationsResponse{
		ScanningStations: pbStations,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetScanningStation(ctx context.Context, req *pb.GetScanningStationRequest) (*pb.GetScanningStationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ss, apiErr := h.scanningStationSvc.GetScanningStation(ctx, domain.GetScanningStationParams{
		ScanningStationID: req.Id,
		Includes:          req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetScanningStationResponse{
		ScanningStation: scanningStationToProto(ss),
	}, nil
}

func (h *gRPCHandler) CreateScanningStation(ctx context.Context, req *pb.CreateScanningStationRequest) (*pb.CreateScanningStationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateScanningStationParams{
		Name:                req.Name,
		Notes:               req.Notes,
		Type:                constants.ScanningStationType(req.Type),
		OperatorRequirement: constants.OperatorRequirement(req.OperatorRequirement),
		DepartmentID:        req.DepartmentId,
		Includes:            req.Includes,
	}

	ss, apiErr := h.scanningStationSvc.CreateScanningStation(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateScanningStationResponse{
		ScanningStation: scanningStationToProto(ss),
	}, nil
}

func (h *gRPCHandler) UpdateScanningStation(ctx context.Context, req *pb.UpdateScanningStationRequest) (*pb.UpdateScanningStationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateScanningStationParams{
		ScanningStationID: req.Id,
		Name:              req.Name,
		Notes:             field.StringClearableFromProto(req.Notes),
		LabelSizeCode:     req.LabelSizeCode,
		LabelTypeCode:     req.LabelTypeCode,
		OperatorRequirement: func() *constants.OperatorRequirement {
			if req.OperatorRequirement == nil {
				return nil
			}
			v := constants.OperatorRequirement(*req.OperatorRequirement)
			return &v
		}(),
		Includes: req.Includes,
	}

	ss, apiErr := h.scanningStationSvc.UpdateScanningStation(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateScanningStationResponse{
		ScanningStation: scanningStationToProto(ss),
	}, nil
}

func (h *gRPCHandler) DeleteScanningStation(ctx context.Context, req *pb.DeleteScanningStationRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.scanningStationSvc.DeleteScanningStation(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) ConnectProductionStepsByScanningStation(ctx context.Context, req *pb.ConnectProductionStepsByScanningStationRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ConnectProductionStepsByNameParams{
		ScanningStationID: req.Id,
		Name:              req.Name,
	}

	apiErr := h.scanningStationSvc.ConnectProductionStepsByName(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
