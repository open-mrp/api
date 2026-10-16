package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type manufacturingMethodGRPCHandler struct {
	pb.UnimplementedCoreManufacturingMethodServiceServer

	svc domain.ManufacturingMethodSvc
}

func RegisterManufacturingMethodService(server *grpc.Server, svc domain.ManufacturingMethodSvc) {
	pb.RegisterCoreManufacturingMethodServiceServer(server, &manufacturingMethodGRPCHandler{svc: svc})
}

func methodPageInfoToProto(pi pagination.PageInfo) *pb.PageInfo {
	return &pb.PageInfo{NextCursor: pi.NextCursor, PrevCursor: pi.PrevCursor, HasNextPage: pi.HasNextPage, HasPrevPage: pi.HasPrevPage}
}

func methodQuantityToProto(q *domain.QuantityValue) *pb.QuantityValue {
	if q == nil {
		return nil
	}
	return &pb.QuantityValue{Value: q.Value, UnitId: q.UnitID, UnitType: q.UnitType}
}

func methodQuantityFromProto(q *pb.MethodQuantityInput) *domain.QuantityValue {
	if q == nil {
		return nil
	}
	return &domain.QuantityValue{Value: q.Value, UnitID: q.UnitId}
}

func methodRateToProto(r *domain.RateValue) *pb.RateInput {
	if r == nil {
		return nil
	}
	return &pb.RateInput{Value: r.Value, UnitId: r.NumeratorUnitID, PerUnitId: r.DenominatorUnitID}
}

func methodRateFromProto(r *pb.RateInput) *domain.RateValue {
	if r == nil {
		return nil
	}
	return &domain.RateValue{Value: r.Value, NumeratorUnitID: r.UnitId, DenominatorUnitID: r.PerUnitId}
}

func methodPriceToProto(p *domain.UnitPrice) *pb.UnitPrice {
	if p == nil {
		return nil
	}
	return &pb.UnitPrice{Amount: p.Amount, Currency: string(p.Currency), PerUnitId: p.PerUnitID}
}

func methodPriceFromProto(p *pb.UnitPrice) *domain.UnitPrice {
	if p == nil {
		return nil
	}
	return &domain.UnitPrice{Amount: p.Amount, Currency: constants.Currency(p.Currency), PerUnitID: p.PerUnitId}
}

func manufacturingMethodToProto(m *domain.ManufacturingMethod) *pb.ManufacturingMethodInfo {
	return &pb.ManufacturingMethodInfo{
		Id:                   m.ID,
		ItemId:               m.ItemID,
		Version:              m.Version,
		Status:               string(m.Status),
		OutputQuantity:       methodQuantityToProto(&m.OutputQuantity),
		SalesOrderLineId:     m.SalesOrderLineID,
		CreatedAt:            timestamppb.New(m.CreatedAt),
		UpdatedAt:            timestamppb.New(m.UpdatedAt),
		OperationIds:         m.OperationIDs,
		OperationsNextCursor: m.OperationsNextCursor,
		MaterialIds:          m.MaterialIDs,
		MaterialsNextCursor:  m.MaterialsNextCursor,
	}
}

func manufacturingMethodsToProto(methods []*domain.ManufacturingMethod) []*pb.ManufacturingMethodInfo {
	out := make([]*pb.ManufacturingMethodInfo, len(methods))
	for i, m := range methods {
		out[i] = manufacturingMethodToProto(m)
	}
	return out
}

func methodOperationToProto(op *domain.MethodOperation) *pb.MethodOperationInfo {
	return &pb.MethodOperationInfo{
		Id:                    op.ID,
		ManufacturingMethodId: op.MethodID,
		Position:              op.Position,
		Sequence:              string(op.Sequence),
		Type:                  string(op.Type),
		WorkCenterId:          op.WorkCenterID,
		MachineIds:            op.MachineIDs,
		ScanningStationId:     op.ScanningStationID,
		SetupTime:             methodQuantityToProto(op.SetupTime),
		LaborTime:             methodRateToProto(op.LaborTime),
		MachineTime:           methodRateToProto(op.MachineTime),
		LaborRate:             methodPriceToProto(op.LaborRate),
		OverheadRate:          methodPriceToProto(op.OverheadRate),
		LevelingPercent:       op.LevelingPercent,
		AllowancePercent:      op.AllowancePercent,
		WorkInstructions:      op.WorkInstructions,
		CreatedAt:             timestamppb.New(op.CreatedAt),
		UpdatedAt:             timestamppb.New(op.UpdatedAt),
	}
}

func methodOperationsToProto(ops []*domain.MethodOperation) []*pb.MethodOperationInfo {
	out := make([]*pb.MethodOperationInfo, len(ops))
	for i, op := range ops {
		out[i] = methodOperationToProto(op)
	}
	return out
}

func methodMaterialToProto(m *domain.MethodMaterial) *pb.MethodMaterialInfo {
	return &pb.MethodMaterialInfo{
		Id:                    m.ID,
		ManufacturingMethodId: m.MethodID,
		ItemId:                m.ItemID,
		OperationId:           m.OperationID,
		Quantity:              methodQuantityToProto(&m.Quantity),
		ScrapQuantity:         methodQuantityToProto(&m.ScrapQuantity),
		Supply:                string(m.Supply),
		WorkInstructions:      m.WorkInstructions,
		CreatedAt:             timestamppb.New(m.CreatedAt),
		UpdatedAt:             timestamppb.New(m.UpdatedAt),
	}
}

func methodMaterialsToProto(materials []*domain.MethodMaterial) []*pb.MethodMaterialInfo {
	out := make([]*pb.MethodMaterialInfo, len(materials))
	for i, m := range materials {
		out[i] = methodMaterialToProto(m)
	}
	return out
}

func methodOperationInputFromProto(in *pb.MethodOperationInput) domain.MethodOperationInput {
	if in == nil {
		return domain.MethodOperationInput{}
	}
	out := domain.MethodOperationInput{
		Position:          in.Position,
		WorkCenterID:      in.WorkCenterId,
		ScanningStationID: fieldpb.StringClearableFromProto(in.ScanningStationId),
		LaborTime:         methodRateFromProto(in.LaborTime),
		LaborRate:         methodPriceFromProto(in.LaborRate),
		OverheadRate:      methodPriceFromProto(in.OverheadRate),
		LevelingPercent:   in.LevelingPercent,
		AllowancePercent:  in.AllowancePercent,
		WorkInstructions:  fieldpb.StringClearableFromProto(in.WorkInstructions),
	}
	if in.Sequence != nil {
		s := constants.OperationSequence(*in.Sequence)
		out.Sequence = &s
	}
	if in.MachineIds != nil {
		ids := []string{}
		if !in.MachineIds.Clear {
			ids = append(ids, in.MachineIds.Value...)
		}
		out.MachineIDs = &ids
	}
	if p := in.SetupTime; p != nil {
		if p.Clear {
			out.SetupTime = field.Clear[domain.QuantityValue]()
		} else {
			out.SetupTime = field.Set(domain.QuantityValue{Value: p.GetValue(), UnitID: p.GetUnitId()})
		}
	}
	if p := in.MachineTime; p != nil {
		if p.Clear || p.Value == nil {
			out.MachineTime = field.Clear[domain.RateValue]()
		} else {
			out.MachineTime = field.Set(*methodRateFromProto(p.Value))
		}
	}
	return out
}

func methodMaterialInputFromProto(in *pb.MethodMaterialInput) domain.MethodMaterialInput {
	if in == nil {
		return domain.MethodMaterialInput{}
	}
	out := domain.MethodMaterialInput{
		ItemID:           in.ItemId,
		OperationID:      in.OperationId,
		Quantity:         methodQuantityFromProto(in.Quantity),
		ScrapQuantity:    methodQuantityFromProto(in.ScrapQuantity),
		WorkInstructions: fieldpb.StringClearableFromProto(in.WorkInstructions),
	}
	if in.Supply != nil {
		s := constants.Supply(*in.Supply)
		out.Supply = &s
	}
	return out
}

func (h *manufacturingMethodGRPCHandler) ListManufacturingMethods(ctx context.Context, req *pb.ListManufacturingMethodsRequest) (*pb.ListManufacturingMethodsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	result, apiErr := h.svc.ListManufacturingMethods(ctx, domain.ListManufacturingMethodsParams{
		Cursor:        req.Cursor,
		Limit:         req.Limit,
		ItemIDs:       req.ItemIds,
		Statuses:      convertStrings[constants.ManufacturingMethodStatus](req.Statuses),
		WorkCenterIDs: req.WorkCenterIds,
		Created:       createdRangeFromProto(req.Created),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ListManufacturingMethodsResponse{
		ManufacturingMethods: manufacturingMethodsToProto(result.Methods),
		PageInfo:             methodPageInfoToProto(result.PageInfo),
	}, nil
}

func (h *manufacturingMethodGRPCHandler) BatchGetManufacturingMethodsByIDs(ctx context.Context, req *pb.BatchGetManufacturingMethodsByIDsRequest) (*pb.BatchGetManufacturingMethodsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	methods, apiErr := h.svc.BatchGetManufacturingMethodsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.BatchGetManufacturingMethodsByIDsResponse{ManufacturingMethods: manufacturingMethodsToProto(methods)}, nil
}

func (h *manufacturingMethodGRPCHandler) CreateManufacturingMethod(ctx context.Context, req *pb.CreateManufacturingMethodRequest) (*pb.ManufacturingMethodResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	m, apiErr := h.svc.CreateManufacturingMethod(ctx, domain.CreateManufacturingMethodParams{
		ItemID:         req.ItemId,
		OutputQuantity: methodQuantityFromProto(req.OutputQuantity),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ManufacturingMethodResponse{ManufacturingMethod: manufacturingMethodToProto(m)}, nil
}

func (h *manufacturingMethodGRPCHandler) UpdateManufacturingMethod(ctx context.Context, req *pb.UpdateManufacturingMethodRequest) (*pb.ManufacturingMethodResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	m, apiErr := h.svc.UpdateManufacturingMethod(ctx, domain.UpdateManufacturingMethodParams{
		MethodID:       req.Id,
		OutputQuantity: methodQuantityFromProto(req.OutputQuantity),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ManufacturingMethodResponse{ManufacturingMethod: manufacturingMethodToProto(m)}, nil
}

func (h *manufacturingMethodGRPCHandler) DeleteManufacturingMethod(ctx context.Context, req *pb.ManufacturingMethodIDRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.svc.DeleteManufacturingMethod(ctx, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &emptypb.Empty{}, nil
}

func (h *manufacturingMethodGRPCHandler) ActivateManufacturingMethod(ctx context.Context, req *pb.ManufacturingMethodIDRequest) (*pb.ManufacturingMethodResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	m, apiErr := h.svc.ActivateManufacturingMethod(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ManufacturingMethodResponse{ManufacturingMethod: manufacturingMethodToProto(m)}, nil
}

func (h *manufacturingMethodGRPCHandler) NewManufacturingMethodVersion(ctx context.Context, req *pb.ManufacturingMethodIDRequest) (*pb.ManufacturingMethodResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	m, apiErr := h.svc.NewManufacturingMethodVersion(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ManufacturingMethodResponse{ManufacturingMethod: manufacturingMethodToProto(m)}, nil
}

func (h *manufacturingMethodGRPCHandler) GetDefaultManufacturingMethodIDs(ctx context.Context, req *pb.GetDefaultManufacturingMethodIDsRequest) (*pb.GetDefaultManufacturingMethodIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ids, apiErr := h.svc.DefaultManufacturingMethodIDs(ctx, req.ItemIds)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.GetDefaultManufacturingMethodIDsResponse{MethodIds: ids}, nil
}

func (h *manufacturingMethodGRPCHandler) GetMethodProductionFlow(ctx context.Context, req *pb.GetMethodProductionFlowRequest) (*pb.GetMethodProductionFlowResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	flow, apiErr := h.svc.GetProductionFlow(ctx, req.ItemId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.GetMethodProductionFlowResponse{ManufacturingMethodIds: flow.MethodIDs}, nil
}

func (h *manufacturingMethodGRPCHandler) ListMethodOperations(ctx context.Context, req *pb.ListMethodMembersRequest) (*pb.ListMethodOperationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	result, apiErr := h.svc.ListMethodOperations(ctx, domain.ListMethodMembersParams{
		MethodID: req.ManufacturingMethodId, Cursor: req.Cursor, Limit: req.Limit,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ListMethodOperationsResponse{
		Operations: methodOperationsToProto(result.Operations),
		PageInfo:   methodPageInfoToProto(result.PageInfo),
	}, nil
}

func (h *manufacturingMethodGRPCHandler) BatchGetMethodOperationsByIDs(ctx context.Context, req *pb.BatchGetMethodOperationsByIDsRequest) (*pb.BatchGetMethodOperationsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ops, apiErr := h.svc.BatchGetMethodOperationsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.BatchGetMethodOperationsByIDsResponse{Operations: methodOperationsToProto(ops)}, nil
}

func (h *manufacturingMethodGRPCHandler) CreateMethodOperation(ctx context.Context, req *pb.CreateMethodOperationRequest) (*pb.MethodOperationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	op, apiErr := h.svc.CreateMethodOperation(ctx, domain.CreateMethodOperationParams{
		MethodID:             req.ManufacturingMethodId,
		MethodOperationInput: methodOperationInputFromProto(req.Operation),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.MethodOperationResponse{Operation: methodOperationToProto(op)}, nil
}

func (h *manufacturingMethodGRPCHandler) UpdateMethodOperation(ctx context.Context, req *pb.UpdateMethodOperationRequest) (*pb.MethodOperationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	op, apiErr := h.svc.UpdateMethodOperation(ctx, domain.UpdateMethodOperationParams{
		MethodID:             req.ManufacturingMethodId,
		OperationID:          req.Id,
		MethodOperationInput: methodOperationInputFromProto(req.Operation),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.MethodOperationResponse{Operation: methodOperationToProto(op)}, nil
}

func (h *manufacturingMethodGRPCHandler) DeleteMethodOperation(ctx context.Context, req *pb.MethodMemberIDRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.svc.DeleteMethodOperation(ctx, req.ManufacturingMethodId, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &emptypb.Empty{}, nil
}

func (h *manufacturingMethodGRPCHandler) ListMethodMaterials(ctx context.Context, req *pb.ListMethodMembersRequest) (*pb.ListMethodMaterialsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	result, apiErr := h.svc.ListMethodMaterials(ctx, domain.ListMethodMembersParams{
		MethodID: req.ManufacturingMethodId, Cursor: req.Cursor, Limit: req.Limit,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ListMethodMaterialsResponse{
		Materials: methodMaterialsToProto(result.Materials),
		PageInfo:  methodPageInfoToProto(result.PageInfo),
	}, nil
}

func (h *manufacturingMethodGRPCHandler) BatchGetMethodMaterialsByIDs(ctx context.Context, req *pb.BatchGetMethodMaterialsByIDsRequest) (*pb.BatchGetMethodMaterialsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	materials, apiErr := h.svc.BatchGetMethodMaterialsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.BatchGetMethodMaterialsByIDsResponse{Materials: methodMaterialsToProto(materials)}, nil
}

func (h *manufacturingMethodGRPCHandler) CreateMethodMaterial(ctx context.Context, req *pb.CreateMethodMaterialRequest) (*pb.MethodMaterialResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	m, apiErr := h.svc.CreateMethodMaterial(ctx, domain.CreateMethodMaterialParams{
		MethodID:            req.ManufacturingMethodId,
		MethodMaterialInput: methodMaterialInputFromProto(req.Material),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.MethodMaterialResponse{Material: methodMaterialToProto(m)}, nil
}

func (h *manufacturingMethodGRPCHandler) UpdateMethodMaterial(ctx context.Context, req *pb.UpdateMethodMaterialRequest) (*pb.MethodMaterialResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	m, apiErr := h.svc.UpdateMethodMaterial(ctx, domain.UpdateMethodMaterialParams{
		MethodID:            req.ManufacturingMethodId,
		MaterialID:          req.Id,
		MethodMaterialInput: methodMaterialInputFromProto(req.Material),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.MethodMaterialResponse{Material: methodMaterialToProto(m)}, nil
}

func (h *manufacturingMethodGRPCHandler) DeleteMethodMaterial(ctx context.Context, req *pb.MethodMemberIDRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	if apiErr := h.svc.DeleteMethodMaterial(ctx, req.ManufacturingMethodId, req.Id); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &emptypb.Empty{}, nil
}
