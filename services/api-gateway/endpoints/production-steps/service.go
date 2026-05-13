package productionstepep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProductionStepSvc interface {
	ListProductionSteps(ctx context.Context, req *ListProductionStepsRequest) (*apiresource.List[apiresource.ProductionStep], *apierror.APIError)
	GetProductionStep(ctx context.Context, req *RetrieveProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError)
	CreateProductionStep(ctx context.Context, req *CreateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError)
	UpdateProductionStep(ctx context.Context, req *UpdateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError)
	DeleteProductionStep(ctx context.Context, req *DeleteProductionStepRequest) (*apiresource.EmptyResource, *apierror.APIError)
	GetProduction(ctx context.Context, req *RetrieveProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError)
	UpdateProduction(ctx context.Context, req *UpdateProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError)
	BulkCreateProductionSteps(ctx context.Context, req *BulkCreateProductionStepsRequest) (*apiresource.BulkCreateProductionStepsResponse, *apierror.APIError)
}

type ProductionStepSvcConfig struct {
	CoreClient pb.CoreProductionStepServiceClient
}

type productionStepSvcImpl struct {
	coreClient pb.CoreProductionStepServiceClient
}

var productionStepSvcTracer = tracing.GetTracer("api-gateway.endpoints.production_steps.service")

func (c *ProductionStepSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("production step endpoint service: core client is required")
	}
	return nil
}

func NewProductionStepSvc(config *ProductionStepSvcConfig) ProductionStepSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &productionStepSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *productionStepSvcImpl) ListProductionSteps(ctx context.Context, req *ListProductionStepsRequest) (*apiresource.List[apiresource.ProductionStep], *apierror.APIError) {
	pbReq := &pb.ListProductionStepsRequest{
		Limit:              req.Limit,
		Cursor:             req.Cursor,
		Query:              req.Query,
		ItemIds:            req.ItemIDs,
		MachineIds:         req.MachineIDs,
		ScanningStationIds: req.ScanningStationIDs,
		InputStepIds:       req.InputStepIDs,
		OutputStepIds:      req.OutputStepIDs,
	}

	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionStepSvcTracer, "service.production_steps.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductionStepsResponse, error) {
			return m.coreClient.ListProductionSteps(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ProductionStepListPresenter(resp), nil
}

func (m *productionStepSvcImpl) GetProductionStep(ctx context.Context, req *RetrieveProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
	pbReq := &pb.GetProductionStepRequest{
		Id: req.ProductionStepID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionStepSvcTracer, "service.production_steps.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductionStepResponse, error) {
			return m.coreClient.GetProductionStep(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductionStepPresenter(resp.ProductionStep)
	return &result, nil
}

func (m *productionStepSvcImpl) CreateProductionStep(ctx context.Context, req *CreateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
	consumptions := make([]*pb.CreateStepConsumptionInput, len(req.Consumptions))
	for i, c := range req.Consumptions {
		consumptions[i] = &pb.CreateStepConsumptionInput{
			ItemId:              c.ItemID,
			QuantityValue:       c.QuantityValue,
			QuantityUnitId:      c.QuantityUnitID,
			WasteQuantityValue:  c.WasteQuantityValue,
			WasteQuantityUnitId: c.WasteQuantityUnitID,
			Instructions:        c.Instructions,
		}
	}

	pbReq := &pb.CreateProductionStepRequest{
		Name:              req.Name,
		Notes:             req.Notes,
		LevelingFactor:    req.LevelingFactor,
		Allowances:        req.Allowances,
		ScanningStationId: req.ScanningStationID,
		DepartmentId:      req.DepartmentID,
		LaborRate: &pb.CreateRateInput{
			Value:             req.LaborRate.Value,
			NumeratorUnitId:   req.LaborRate.NumeratorUnitID,
			DenominatorUnitId: req.LaborRate.DenominatorUnitID,
		},
		LaborTime: &pb.CreateRateInput{
			Value:             req.LaborTime.Value,
			NumeratorUnitId:   req.LaborTime.NumeratorUnitID,
			DenominatorUnitId: req.LaborTime.DenominatorUnitID,
		},
		OverheadRate: &pb.CreateRateInput{
			Value:             req.OverheadRate.Value,
			NumeratorUnitId:   req.OverheadRate.NumeratorUnitID,
			DenominatorUnitId: req.OverheadRate.DenominatorUnitID,
		},
		Production: &pb.CreateProductionInput{
			ItemId:         req.Production.ItemID,
			QuantityValue:  req.Production.QuantityValue,
			QuantityUnitId: req.Production.QuantityUnitID,
		},
		Consumptions: consumptions,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionStepSvcTracer, "service.production_steps.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateProductionStepResponse, error) {
			return m.coreClient.CreateProductionStep(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductionStepPresenter(resp.ProductionStep)
	return &result, nil
}

func (m *productionStepSvcImpl) UpdateProductionStep(ctx context.Context, req *UpdateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
	pbReq := &pb.UpdateProductionStepRequest{
		Id:                req.ProductionStepID,
		Name:              req.Name,
		LevelingFactor:    req.LevelingFactor,
		Allowances:        req.Allowances,
		ScanningStationId: req.ScanningStationID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionStepSvcTracer, "service.production_steps.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductionStepResponse, error) {
			return m.coreClient.UpdateProductionStep(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductionStepPresenter(resp.ProductionStep)
	return &result, nil
}

func (m *productionStepSvcImpl) DeleteProductionStep(ctx context.Context, req *DeleteProductionStepRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteProductionStepRequest{
		Id: req.ProductionStepID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, productionStepSvcTracer, "service.production_steps.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteProductionStepResponse, error) {
			return m.coreClient.DeleteProductionStep(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *productionStepSvcImpl) GetProduction(ctx context.Context, req *RetrieveProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError) {
	pbReq := &pb.GetProductionRequest{
		ProductionStepId: req.ProductionStepID,
		Id:               req.ProductionID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionStepSvcTracer, "service.production_steps.get_production", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductionResponse, error) {
			return m.coreClient.GetProduction(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := productionOutputPresenter(resp.Production)
	return result, nil
}

func (m *productionStepSvcImpl) UpdateProduction(ctx context.Context, req *UpdateProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError) {
	pbReq := &pb.UpdateProductionRequest{
		ProductionStepId: req.ProductionStepID,
		Id:               req.ProductionID,
		ItemId:           req.ItemID,
		QuantityValue:    req.QuantityValue,
		QuantityUnitId:   req.QuantityUnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionStepSvcTracer, "service.production_steps.update_production", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductionResponse, error) {
			return m.coreClient.UpdateProduction(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := productionOutputPresenter(resp.Production)
	return result, nil
}

func (m *productionStepSvcImpl) BulkCreateProductionSteps(ctx context.Context, req *BulkCreateProductionStepsRequest) (*apiresource.BulkCreateProductionStepsResponse, *apierror.APIError) {
	pbSteps := make([]*pb.BulkCreateProductionStepInput, len(req.Steps))
	for i, step := range req.Steps {
		consumptions := make([]*pb.BulkCreateConsumptionInput, len(step.Consumptions))
		for j, c := range step.Consumptions {
			consumptions[j] = &pb.BulkCreateConsumptionInput{
				Sku:          c.SKU,
				Measure:      fmt.Sprintf("%g", c.Measure),
				Instructions: c.Instructions,
			}
		}

		productions := make([]*pb.BulkCreateProductionInput, len(step.Productions))
		for j, p := range step.Productions {
			productions[j] = &pb.BulkCreateProductionInput{
				Sku:     p.SKU,
				Measure: fmt.Sprintf("%g", p.Measure),
			}
		}

		var allowances, levelingFactor *string
		if step.Allowances != nil {
			s := fmt.Sprintf("%g", *step.Allowances)
			allowances = &s
		}
		if step.LevelingFactor != nil {
			s := fmt.Sprintf("%g", *step.LevelingFactor)
			levelingFactor = &s
		}

		pbSteps[i] = &pb.BulkCreateProductionStepInput{
			Name:           step.Name,
			Consumptions:   consumptions,
			Productions:    productions,
			LaborRate:      fmt.Sprintf("%g", step.LaborRate),
			LaborTime:      fmt.Sprintf("%g", step.LaborTime),
			LaborTimeUnit:  step.LaborTimeUnit,
			OverheadRate:   fmt.Sprintf("%g", step.OverheadRate),
			Allowances:     allowances,
			LevelingFactor: levelingFactor,
			Station:        step.Station,
		}
	}

	pbReq := &pb.BulkCreateProductionStepsRequest{
		Steps: pbSteps,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionStepSvcTracer, "service.production_steps.bulk_create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkCreateProductionStepsResponse, error) {
			return m.coreClient.BulkCreateProductionSteps(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	results := make([]apiresource.BulkCreateProductionStepResult, len(resp.Results))
	for i, r := range resp.Results {
		status := "failed"
		if r.Success {
			status = "created"
		}
		results[i] = apiresource.BulkCreateProductionStepResult{
			Name:             r.Name,
			Status:           status,
			Error:            r.Error,
			ProductionStepID: r.ProductionStepId,
			Action:           r.Action,
		}
	}

	return &apiresource.BulkCreateProductionStepsResponse{
		Object: "list",
		Data:   results,
	}, nil
}
