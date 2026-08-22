package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type productionStepGRPCHandler struct {
	pb.UnimplementedCoreProductionStepServiceServer
	productionStepSvc domain.ProductionStepSvc
	productionSvc     domain.ProductionSvc
}

func RegisterProductionStepService(server *grpc.Server, productionStepSvc domain.ProductionStepSvc, productionSvc domain.ProductionSvc) {
	h := &productionStepGRPCHandler{
		productionStepSvc: productionStepSvc,
		productionSvc:     productionSvc,
	}
	pb.RegisterCoreProductionStepServiceServer(server, h)
}

func productionStepRateToProto(r *domain.ProductionStepRate) *pb.ProductionStepRateInfo {
	if r == nil {
		return nil
	}
	return &pb.ProductionStepRateInfo{
		Id:                          r.ID,
		Value:                       r.Value,
		NumeratorUnitId:             r.NumeratorUnit.ID,
		NumeratorUnitAbbreviation:   r.NumeratorUnit.Abbreviation,
		NumeratorUnitType:           r.NumeratorUnit.Type,
		DenominatorUnitId:           r.DenominatorUnit.ID,
		DenominatorUnitAbbreviation: r.DenominatorUnit.Abbreviation,
		DenominatorUnitType:         r.DenominatorUnit.Type,
	}
}

func productionToProto(p *domain.Production) *pb.ProductionInfo {
	if p == nil {
		return nil
	}
	info := &pb.ProductionInfo{
		Id:               p.ID,
		ItemId:           p.ItemID,
		ItemSku:          p.ItemSKU,
		ItemTypeCode:     p.ItemTypeCode,
		Quantity:         quantityToProto(&p.Quantity),
		ProductionStepId: p.ProductionStepID,
		CreatedAt:        timestamppb.New(p.CreatedAt),
		UpdatedAt:        timestamppb.New(p.UpdatedAt),
	}
	if p.ItemDescription != nil {
		info.ItemDescription = p.ItemDescription
	}
	return info
}

func productionStepToProto(s *domain.ProductionStep) *pb.ProductionStepInfo {
	if s == nil {
		return nil
	}

	info := &pb.ProductionStepInfo{
		Id:             s.ID,
		Name:           s.Name,
		LevelingFactor: s.LevelingFactor,
		Allowances:     s.Allowances,
		LaborRate:      productionStepRateToProto(s.LaborRate),
		LaborTime:      productionStepRateToProto(s.LaborTime),
		OverheadRate:   productionStepRateToProto(s.OverheadRate),
		Production:     productionToProto(s.Production),
		CreatedAt:      timestamppb.New(s.CreatedAt),
		UpdatedAt:      timestamppb.New(s.UpdatedAt),
	}

	if s.Notes != nil {
		info.Notes = s.Notes
	}

	if s.DepartmentID != nil {
		info.DepartmentId = s.DepartmentID
	}

	// Consumptions
	consumptions := make([]*pb.ConsumptionInfo, len(s.Consumptions))
	for i, c := range s.Consumptions {
		consumptions[i] = consumptionToProto(&c)
	}
	info.Consumptions = consumptions

	// Machines
	machines := make([]*pb.LightMachineInfo, len(s.Machines))
	for i, m := range s.Machines {
		machines[i] = &pb.LightMachineInfo{Id: m.ID, Name: m.Name}
	}
	info.Machines = machines

	// Scanning station
	if s.ScanningStation != nil {
		info.ScanningStation = &pb.LightScanningStationInfo{
			Id:   s.ScanningStation.ID,
			Name: s.ScanningStation.Name,
		}
	}

	// In/Out steps
	inSteps := make([]*pb.LightProductionStepInfo, len(s.InSteps))
	for i, st := range s.InSteps {
		inSteps[i] = &pb.LightProductionStepInfo{Id: st.ID, Name: st.Name}
	}
	info.InSteps = inSteps

	outSteps := make([]*pb.LightProductionStepInfo, len(s.OutSteps))
	for i, st := range s.OutSteps {
		outSteps[i] = &pb.LightProductionStepInfo{Id: st.ID, Name: st.Name}
	}
	info.OutSteps = outSteps

	return info
}

func (h *productionStepGRPCHandler) ExportProductionSteps(ctx context.Context, req *pb.ExportProductionStepsRequest) (*pb.ExportProductionStepsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	job, apiErr := h.productionStepSvc.ExportProductionSteps(ctx, domain.ExportProductionStepsParams{Query: req.Query})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ExportProductionStepsResponse{Job: jobToProto(job)}, nil
}

func (h *productionStepGRPCHandler) ListProductionSteps(ctx context.Context, req *pb.ListProductionStepsRequest) (*pb.ListProductionStepsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListProductionStepsParams{
		Limit:              req.Limit,
		Cursor:             req.Cursor,
		Query:              req.Query,
		ItemIDs:            req.ItemIds,
		MachineIDs:         req.MachineIds,
		ScanningStationIDs: req.ScanningStationIds,
		InputStepIDs:       req.InputStepIds,
		OutputStepIDs:      req.OutputStepIds,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.productionStepSvc.ListProductionSteps(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	steps := make([]*pb.ProductionStepInfo, len(result.Steps))
	for i, s := range result.Steps {
		steps[i] = productionStepToProto(s)
	}

	return &pb.ListProductionStepsResponse{
		ProductionSteps: steps,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *productionStepGRPCHandler) GetProductionStep(ctx context.Context, req *pb.GetProductionStepRequest) (*pb.GetProductionStepResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	step, apiErr := h.productionStepSvc.GetProductionStep(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetProductionStepResponse{
		ProductionStep: productionStepToProto(step),
	}, nil
}

func (h *productionStepGRPCHandler) CreateProductionStep(ctx context.Context, req *pb.CreateProductionStepRequest) (*pb.CreateProductionStepResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	consumptions := make([]domain.CreateStepConsumptionParams, len(req.Consumptions))
	for i, c := range req.Consumptions {
		consumptions[i] = domain.CreateStepConsumptionParams{
			ItemID:              c.ItemId,
			QuantityValue:       c.QuantityValue,
			QuantityUnitID:      c.QuantityUnitId,
			WasteQuantityValue:  c.WasteQuantityValue,
			WasteQuantityUnitID: c.WasteQuantityUnitId,
			Instructions:        c.Instructions,
		}
	}

	step, apiErr := h.productionStepSvc.CreateProductionStep(ctx, domain.CreateProductionStepParams{
		Name:              req.Name,
		Notes:             req.Notes,
		LevelingFactor:    req.LevelingFactor,
		Allowances:        req.Allowances,
		ScanningStationID: req.ScanningStationId,
		DepartmentID:      req.DepartmentId,
		LaborRate: domain.CreateRateParams{
			Value:             req.LaborRate.Value,
			NumeratorUnitID:   req.LaborRate.NumeratorUnitId,
			DenominatorUnitID: req.LaborRate.DenominatorUnitId,
		},
		LaborTime: domain.CreateRateParams{
			Value:             req.LaborTime.Value,
			NumeratorUnitID:   req.LaborTime.NumeratorUnitId,
			DenominatorUnitID: req.LaborTime.DenominatorUnitId,
		},
		OverheadRate: domain.CreateRateParams{
			Value:             req.OverheadRate.Value,
			NumeratorUnitID:   req.OverheadRate.NumeratorUnitId,
			DenominatorUnitID: req.OverheadRate.DenominatorUnitId,
		},
		Production: domain.CreateProductionParams{
			ItemID:         req.Production.ItemId,
			QuantityValue:  req.Production.QuantityValue,
			QuantityUnitID: req.Production.QuantityUnitId,
		},
		Consumptions: consumptions,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateProductionStepResponse{
		ProductionStep: productionStepToProto(step),
	}, nil
}

func (h *productionStepGRPCHandler) UpdateProductionStep(ctx context.Context, req *pb.UpdateProductionStepRequest) (*pb.UpdateProductionStepResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	step, apiErr := h.productionStepSvc.UpdateProductionStep(ctx, domain.UpdateProductionStepParams{
		ProductionStepID:  req.Id,
		Name:              req.Name,
		LevelingFactor:    req.LevelingFactor,
		Allowances:        req.Allowances,
		ScanningStationID: req.ScanningStationId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateProductionStepResponse{
		ProductionStep: productionStepToProto(step),
	}, nil
}

func (h *productionStepGRPCHandler) DeleteProductionStep(ctx context.Context, req *pb.DeleteProductionStepRequest) (*pb.DeleteProductionStepResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.productionStepSvc.DeleteProductionStep(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteProductionStepResponse{}, nil
}

func (h *productionStepGRPCHandler) GetProduction(ctx context.Context, req *pb.GetProductionRequest) (*pb.GetProductionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	production, apiErr := h.productionSvc.GetProduction(ctx, req.ProductionStepId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetProductionResponse{
		Production: productionToProto(production),
	}, nil
}

func (h *productionStepGRPCHandler) BulkUpsertProductionSteps(ctx context.Context, req *pb.BulkUpsertProductionStepsRequest) (*pb.BulkUpsertProductionStepsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	rateFromProto := func(r *pb.UpsertRateInput) domain.UpsertRateParams {
		if r == nil {
			return domain.UpsertRateParams{}
		}
		return domain.UpsertRateParams{
			Value:           r.Value,
			NumeratorUnit:   unitIdentifierFromProto(r.NumeratorUnit),
			DenominatorUnit: unitIdentifierFromProto(r.DenominatorUnit),
		}
	}

	steps := make([]domain.UpsertProductionStepParams, len(req.ProductionSteps))
	for i, s := range req.ProductionSteps {
		if s == nil {
			return nil, contracts.NewMissingGRPCRequestDataError()
		}

		var production domain.UpsertProductionParams
		if s.Production != nil {
			production = domain.UpsertProductionParams{
				Item:          itemIdentifierFromProto(s.Production.Item),
				QuantityValue: s.Production.QuantityValue,
				QuantityUnit:  unitIdentifierFromProto(s.Production.QuantityUnit),
			}
		}

		consumptions := make([]domain.UpsertStepConsumptionParams, len(s.Consumptions))
		for j, c := range s.Consumptions {
			consumptions[j] = domain.UpsertStepConsumptionParams{
				Item:               itemIdentifierFromProto(c.Item),
				QuantityValue:      c.QuantityValue,
				QuantityUnit:       unitIdentifierFromProto(c.QuantityUnit),
				WasteQuantityValue: c.WasteQuantityValue,
				WasteQuantityUnit:  unitIdentifierPtrFromProto(c.WasteQuantityUnit),
				Instructions:       c.Instructions,
			}
		}

		steps[i] = domain.UpsertProductionStepParams{
			Name:            s.Name,
			Notes:           s.Notes,
			LevelingFactor:  s.LevelingFactor,
			Allowances:      s.Allowances,
			ScanningStation: objectIdentifierPtrFromProto(s.ScanningStation),
			Department:      objectIdentifierPtrFromProto(s.Department),
			LaborRate:       rateFromProto(s.LaborRate),
			LaborTime:       rateFromProto(s.LaborTime),
			OverheadRate:    rateFromProto(s.OverheadRate),
			Production:      production,
			Consumptions:    consumptions,
		}
	}

	job, apiErr := h.productionStepSvc.BulkUpsertProductionSteps(ctx, domain.BulkUpsertProductionStepsParams{ProductionSteps: steps})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.BulkUpsertProductionStepsResponse{Job: jobToProto(job)}, nil
}

func (h *productionStepGRPCHandler) BulkCreateProductionSteps(ctx context.Context, req *pb.BulkCreateProductionStepsRequest) (*pb.BulkCreateProductionStepsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	steps := make([]domain.BulkCreateProductionStepInput, len(req.Steps))
	for i, s := range req.Steps {
		consumptions := make([]domain.BulkCreateConsumptionInput, len(s.Consumptions))
		for j, c := range s.Consumptions {
			consumptions[j] = domain.BulkCreateConsumptionInput{
				SKU:          c.Sku,
				Measure:      c.Measure,
				Instructions: c.Instructions,
			}
		}

		productions := make([]domain.BulkCreateProductionInput, len(s.Productions))
		for j, p := range s.Productions {
			productions[j] = domain.BulkCreateProductionInput{
				SKU:     p.Sku,
				Measure: p.Measure,
			}
		}

		steps[i] = domain.BulkCreateProductionStepInput{
			Name:           s.Name,
			Consumptions:   consumptions,
			Productions:    productions,
			LaborRate:      s.LaborRate,
			LaborTime:      s.LaborTime,
			LaborTimeUnit:  s.LaborTimeUnit,
			OverheadRate:   s.OverheadRate,
			Allowances:     s.Allowances,
			LevelingFactor: s.LevelingFactor,
			Station:        s.Station,
		}
	}

	results, apiErr := h.productionStepSvc.BulkCreateProductionSteps(ctx, domain.BulkCreateProductionStepsParams{
		Steps: steps,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbResults := make([]*pb.BulkCreateProductionStepResult, len(results))
	for i, r := range results {
		pbResults[i] = &pb.BulkCreateProductionStepResult{
			Name:             r.Name,
			Success:          r.Success,
			Error:            r.Error,
			ProductionStepId: r.ProductionStepID,
			Action:           r.Action,
		}
	}

	return &pb.BulkCreateProductionStepsResponse{
		Results: pbResults,
	}, nil
}

func (h *productionStepGRPCHandler) UpdateProduction(ctx context.Context, req *pb.UpdateProductionRequest) (*pb.UpdateProductionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	production, apiErr := h.productionSvc.UpdateProduction(ctx, domain.UpdateProductionParams{
		ProductionStepID: req.ProductionStepId,
		ProductionID:     req.Id,
		ItemID:           req.ItemId,
		QuantityValue:    req.QuantityValue,
		QuantityUnitID:   req.QuantityUnitId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateProductionResponse{
		Production: productionToProto(production),
	}, nil
}
