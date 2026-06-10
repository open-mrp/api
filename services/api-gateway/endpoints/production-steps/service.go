package productionstepep

import (
	"context"
	"fmt"

	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
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
	// CoreClient (required) is the core-service production-step gRPC client.
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

	meta := resourcekit.GetLoadMeta(ctx)
	steps := make([]apiresource.ProductionStep, len(resp.ProductionSteps))
	for i, s := range resp.ProductionSteps {
		steps[i] = productionStepFromProto(s)
		stashProductionStepMeta(meta, s)
	}

	return apiresource.NewList(steps, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
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

	meta := resourcekit.GetLoadMeta(ctx)
	result := productionStepFromProto(resp.ProductionStep)
	stashProductionStepMeta(meta, resp.ProductionStep)
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
			Instructions:        c.Instructions.Ptr(),
		}
	}

	pbReq := &pb.CreateProductionStepRequest{
		Name:              req.Name,
		Notes:             req.Notes.Ptr(),
		LevelingFactor:    req.LevelingFactor,
		Allowances:        req.Allowances,
		ScanningStationId: req.ScanningStationID.Ptr(),
		DepartmentId:      req.DepartmentID.Ptr(),
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

	meta := resourcekit.GetLoadMeta(ctx)
	result := productionStepFromProto(resp.ProductionStep)
	stashProductionStepMeta(meta, resp.ProductionStep)
	return &result, nil
}

func (m *productionStepSvcImpl) UpdateProductionStep(ctx context.Context, req *UpdateProductionStepRequest) (*apiresource.ProductionStep, *apierror.APIError) {
	pbReq := &pb.UpdateProductionStepRequest{
		Id:                req.ProductionStepID,
		Name:              req.Name.Ptr(),
		LevelingFactor:    req.LevelingFactor.Ptr(),
		Allowances:        req.Allowances.Ptr(),
		ScanningStationId: req.ScanningStationID.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionStepSvcTracer, "service.production_steps.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductionStepResponse, error) {
			return m.coreClient.UpdateProductionStep(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := productionStepFromProto(resp.ProductionStep)
	stashProductionStepMeta(meta, resp.ProductionStep)
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

	meta := resourcekit.GetLoadMeta(ctx)
	result := productionOutputFromProto(resp.Production)
	stashProductionOutputMeta(meta, resp.Production)
	return result, nil
}

func (m *productionStepSvcImpl) UpdateProduction(ctx context.Context, req *UpdateProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError) {
	pbReq := &pb.UpdateProductionRequest{
		ProductionStepId: req.ProductionStepID,
		Id:               req.ProductionID,
		ItemId:           req.ItemID.Ptr(),
		QuantityValue:    req.QuantityValue.Ptr(),
		QuantityUnitId:   req.QuantityUnitID.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productionStepSvcTracer, "service.production_steps.update_production", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductionResponse, error) {
			return m.coreClient.UpdateProduction(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := productionOutputFromProto(resp.Production)
	stashProductionOutputMeta(meta, resp.Production)
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
				Instructions: c.Instructions.Ptr(),
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
		if v, ok := step.Allowances.Value(); ok {
			s := fmt.Sprintf("%g", v)
			allowances = &s
		}
		if v, ok := step.LevelingFactor.Value(); ok {
			s := fmt.Sprintf("%g", v)
			levelingFactor = &s
		}

		pbSteps[i] = &pb.BulkCreateProductionStepInput{
			Name:           step.Name,
			Consumptions:   consumptions,
			Productions:    productions,
			LaborRate:      fmt.Sprintf("%g", step.LaborRate),
			LaborTime:      fmt.Sprintf("%g", step.LaborTime),
			LaborTimeUnit:  step.LaborTimeUnit.Ptr(),
			OverheadRate:   fmt.Sprintf("%g", step.OverheadRate),
			Allowances:     allowances,
			LevelingFactor: levelingFactor,
			Station:        step.Station.Ptr(),
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

var embeddedRateTimestamp = time.Unix(1, 0).UTC()

func productionStepFromProto(s *pb.ProductionStepInfo) apiresource.ProductionStep {
	if s == nil {
		return apiresource.ProductionStep{}
	}

	return apiresource.ProductionStep{
		ID:             s.Id,
		Object:         constants.ObjectTypeProductionStep,
		Name:           s.Name,
		Notes:          s.Notes,
		LevelingFactor: s.LevelingFactor,
		Allowances:     s.Allowances,
		LaborRate:      rateFromStepProto(s.LaborRate),
		LaborTime:      rateFromStepProto(s.LaborTime),
		OverheadRate:   rateFromStepProto(s.OverheadRate),
		CreatedAt:      grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(s.UpdatedAt),
	}
}

func rateFromStepProto(r *pb.ProductionStepRateInfo) *apiresource.Rate {
	if r == nil {
		return nil
	}
	return &apiresource.Rate{
		ID:     r.Id,
		Object: constants.ObjectTypeRate,
		Value:  r.Value,
		// numerator_unit / denominator_unit left nil: expandable, loaded with real
		// data via ?include=; never fabricated. display_value carries the rate.
		DisplayValue: apiresource.FormatRateDisplayValue(r.Value, r.NumeratorUnitAbbreviation, r.NumeratorUnitType, r.DenominatorUnitAbbreviation),
		CreatedAt:    embeddedRateTimestamp,
		UpdatedAt:    embeddedRateTimestamp,
	}
}

func quantityFromStepProto(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	norm := apiresource.NormalizeQuantityValue(q.Value, q.UnitType)
	return &apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        norm,
		DisplayValue: apiresource.FormatDisplayValue(norm, q.UnitAbbreviation, q.UnitType),
	}
}

func productionOutputFromProto(p *pb.ProductionInfo) *apiresource.ProductionOutput {
	if p == nil {
		return nil
	}

	return &apiresource.ProductionOutput{
		ID:        p.Id,
		Object:    constants.ObjectTypeProduction,
		Quantity:  quantityFromStepProto(p.Quantity),
		CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
	}
}

func stashProductionOutputMeta(meta *resourcekit.LoadMeta, p *pb.ProductionInfo) {
	if p == nil || p.ItemId == "" {
		return
	}
	itemTS := grpcutil.TimestampToTime(p.CreatedAt)
	meta.Set(constants.ObjectTypeProduction, p.Id, "produced_item", &apiresource.Item{
		ID:           p.ItemId,
		Object:       constants.ObjectTypeItem,
		SKU:          p.ItemSku,
		Description:  p.ItemDescription,
		ItemTypeCode: constants.ItemTypeCode(p.ItemTypeCode),
		CreatedAt:    itemTS,
		UpdatedAt:    itemTS,
	})
}

func stashProductionStepMeta(meta *resourcekit.LoadMeta, s *pb.ProductionStepInfo) {
	if s == nil {
		return
	}

	stepTS := grpcutil.TimestampToTime(s.CreatedAt)

	if s.Production != nil {
		prod := productionOutputFromProto(s.Production)
		if s.Production.ItemId != "" {
			itemTS := grpcutil.TimestampToTime(s.Production.CreatedAt)
			prod.ProducedItem = &apiresource.Item{
				ID:           s.Production.ItemId,
				Object:       constants.ObjectTypeItem,
				SKU:          s.Production.ItemSku,
				Description:  s.Production.ItemDescription,
				ItemTypeCode: constants.ItemTypeCode(s.Production.ItemTypeCode),
				CreatedAt:    itemTS,
				UpdatedAt:    itemTS,
			}
		}
		meta.Set(constants.ObjectTypeProductionStep, s.Id, "production", prod)
	}

	consumptions := make([]apiresource.Consumption, len(s.Consumptions))
	for i, c := range s.Consumptions {
		consumptions[i] = stepConsumptionFromProto(c)
	}
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "consumptions", apiresource.NewList(consumptions, apiresource.PageInfo{}))

	machines := make([]apiresource.Machine, len(s.Machines))
	for i, m := range s.Machines {
		sn := m.GetSerialNumber()
		if sn == "" {
			sn = "—"
		}
		mCreated := stepTS
		if m.CreatedAt != nil {
			mCreated = grpcutil.TimestampToTime(m.CreatedAt)
		}
		mUpdated := stepTS
		if m.UpdatedAt != nil {
			mUpdated = grpcutil.TimestampToTime(m.UpdatedAt)
		}
		machines[i] = apiresource.Machine{
			ID:           m.Id,
			Object:       constants.ObjectTypeMachine,
			Name:         m.Name,
			SerialNumber: sn,
			CreatedAt:    mCreated,
			UpdatedAt:    mUpdated,
		}
	}
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "machines", apiresource.NewList(machines, apiresource.PageInfo{}))

	if s.ScanningStation != nil {
		ss := s.ScanningStation
		ssType := constants.ScanningStationType(ss.Type)
		if !ssType.IsValid() {
			ssType = constants.ScanningStationTypeInitBatch
		}
		ssCreated := stepTS
		if ss.CreatedAt != nil {
			ssCreated = grpcutil.TimestampToTime(ss.CreatedAt)
		}
		ssUpdated := stepTS
		if ss.UpdatedAt != nil {
			ssUpdated = grpcutil.TimestampToTime(ss.UpdatedAt)
		}
		meta.Set(constants.ObjectTypeProductionStep, s.Id, "scanning_station", &apiresource.ScanningStation{
			ID:                  ss.Id,
			Object:              constants.ObjectTypeScanningStation,
			Name:                ss.Name,
			Type:                ssType,
			OperatorRequirement: constants.OperatorRequirementNone,
			CreatedAt:           ssCreated,
			UpdatedAt:           ssUpdated,
		})
	}

	if s.DepartmentId != nil && *s.DepartmentId != "" {
		meta.Set(constants.ObjectTypeProductionStep, s.Id, "department", &apiresource.Department{
			ID:        *s.DepartmentId,
			Object:    constants.ObjectTypeDepartment,
			Name:      "Department",
			CreatedAt: stepTS,
			UpdatedAt: stepTS,
		})
	}

	inSteps := make([]apiresource.ProductionStep, len(s.InSteps))
	for i, st := range s.InSteps {
		inSteps[i] = lightProductionStepToResource(st, stepTS)
	}
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "in_steps", apiresource.NewList(inSteps, apiresource.PageInfo{}))

	outSteps := make([]apiresource.ProductionStep, len(s.OutSteps))
	for i, st := range s.OutSteps {
		outSteps[i] = lightProductionStepToResource(st, stepTS)
	}
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "out_steps", apiresource.NewList(outSteps, apiresource.PageInfo{}))
}

func stepConsumptionFromProto(c *pb.ConsumptionInfo) apiresource.Consumption {
	if c == nil {
		return apiresource.Consumption{}
	}

	itemTS := grpcutil.TimestampToTime(c.CreatedAt)
	var consumedItem *apiresource.Item
	if c.ItemId != "" {
		consumedItem = &apiresource.Item{
			ID:           c.ItemId,
			Object:       constants.ObjectTypeItem,
			SKU:          c.ItemSku,
			Description:  c.ItemDescription,
			ItemTypeCode: constants.ItemTypeCode(c.ItemTypeCode),
			CreatedAt:    itemTS,
			UpdatedAt:    itemTS,
		}
	}

	return apiresource.Consumption{
		ID:            c.Id,
		Object:        constants.ObjectTypeConsumption,
		Quantity:      quantityFromStepProto(c.Quantity),
		WasteQuantity: quantityFromStepProto(c.WasteQuantity),
		ConsumedItem:  consumedItem,
		Instructions:  c.Instructions,
		CreatedAt:     itemTS,
		UpdatedAt:     grpcutil.TimestampToTime(c.UpdatedAt),
	}
}

// lightProductionStepToResource maps the real, lightweight in/out-step data the
// production-step proto carries inline into a ProductionStep reference. It never
// fabricates identifiers; absent optional decimals default to their mathematical
// identity (leveling_factor 1, allowances 0) so the resource stays valid.
func lightProductionStepToResource(st *pb.LightProductionStepInfo, fallback time.Time) apiresource.ProductionStep {
	lf := st.GetLevelingFactor()
	if lf == "" {
		lf = "1"
	}
	al := st.GetAllowances()
	if al == "" {
		al = "0"
	}
	ca := fallback
	if st.CreatedAt != nil {
		ca = grpcutil.TimestampToTime(st.CreatedAt)
	}
	ua := fallback
	if st.UpdatedAt != nil {
		ua = grpcutil.TimestampToTime(st.UpdatedAt)
	}
	return apiresource.ProductionStep{
		ID:             st.Id,
		Object:         constants.ObjectTypeProductionStep,
		Name:           st.Name,
		LevelingFactor: lf,
		Allowances:     al,
		CreatedAt:      ca,
		UpdatedAt:      ua,
	}
}
