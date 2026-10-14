package productionflowep

import (
	"context"
	"time"

	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/include"

	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func flowQuantityPresenter(q *pb.QuantityInfo) *apiresource.Quantity {
	return resourceloaders.QuantityFromProto(q)
}

func flowProductionPresenter(ctx context.Context, p *pb.ProductionFlowProductionInfo, ts time.Time) *apiresource.ProductionFlowProduction {
	if p == nil {
		return nil
	}
	if p.ItemId != "" {
		// produced_item is expandable: stash only the FK id; LoadItems fetches the
		// real Item on ?include=...produced_item. Never fabricate.
		include.GetLoadMeta(ctx).Set(constants.ObjectTypeProduction, p.Id, "produced_item_id", p.ItemId)
	}
	createdAt := ts
	updatedAt := ts
	if p.Quantity != nil {
		if qCreated := grpcutil.TimestampToTime(p.Quantity.CreatedAt); !qCreated.IsZero() {
			createdAt = qCreated
		}
		if qUpdated := grpcutil.TimestampToTime(p.Quantity.UpdatedAt); !qUpdated.IsZero() {
			updatedAt = qUpdated
		}
	}
	return &apiresource.ProductionFlowProduction{
		ID:        p.Id,
		Object:    constants.ObjectTypeProduction,
		Quantity:  flowQuantityPresenter(p.Quantity),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func flowConsumptionPresenter(ctx context.Context, c *pb.ProductionFlowConsumptionInfo, ts time.Time) apiresource.ProductionFlowConsumption {
	if c == nil {
		return apiresource.ProductionFlowConsumption{}
	}
	if c.ItemId != "" {
		// consumed_item is expandable: stash only the FK id; LoadItems fetches the
		// real Item on ?include=...consumed_item. Never fabricate.
		include.GetLoadMeta(ctx).Set(constants.ObjectTypeConsumption, c.Id, "consumed_item_id", c.ItemId)
	}
	createdAt := grpcutil.TimestampToTime(c.CreatedAt)
	if createdAt.IsZero() {
		createdAt = ts
	}
	updatedAt := grpcutil.TimestampToTime(c.UpdatedAt)
	if updatedAt.IsZero() {
		updatedAt = ts
	}
	return apiresource.ProductionFlowConsumption{
		ID:            c.Id,
		Object:        constants.ObjectTypeConsumption,
		Quantity:      flowQuantityPresenter(c.Quantity),
		WasteQuantity: flowQuantityPresenter(c.WasteQuantity),
		Instructions:  c.Instructions,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

// linkedStepFromProto presents a flow step as the production step another flow step links to.
func linkedStepFromProto(s *pb.ProductionFlowStepInfo) apiresource.ProductionStep {
	var notes *string
	if s.Notes != nil && *s.Notes != "" {
		notes = s.Notes
	}
	return apiresource.ProductionStep{
		ID:             s.Id,
		Object:         constants.ObjectTypeProductionStep,
		Name:           s.Name,
		Notes:          notes,
		LevelingFactor: s.LevelingFactor,
		Allowances:     s.Allowances,
		LaborRate:      resourceloaders.UnitPriceFromProto(s.LaborRate),
		LaborTime:      resourceloaders.RateFromProto(s.LaborTime),
		OverheadRate:   resourceloaders.UnitPriceFromProto(s.OverheadRate),
		CreatedAt:      grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(s.UpdatedAt),
	}
}

func productionFlowStepFromProto(ctx context.Context, s *pb.ProductionFlowStepInfo, flowSteps map[string]*pb.ProductionFlowStepInfo) apiresource.ProductionFlowStep {
	if s == nil {
		return apiresource.ProductionFlowStep{}
	}

	stubTS := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	meta := include.GetLoadMeta(ctx)

	production := flowProductionPresenter(ctx, s.Production, stubTS)
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "production", production)

	consumptionItems := make([]apiresource.ProductionFlowConsumption, 0, len(s.Consumptions))
	for _, c := range s.Consumptions {
		consumptionItems = append(consumptionItems, flowConsumptionPresenter(ctx, c, stubTS))
	}
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "consumptions",
		apiresource.NewList(consumptionItems, apiresource.PageInfo{}))

	// A flow's in and out steps are restricted to steps in the flow, so each is one of its own steps.
	linked := func(ids []string) *apiresource.List[apiresource.ProductionStep] {
		items := make([]apiresource.ProductionStep, 0, len(ids))
		for _, id := range ids {
			if step, ok := flowSteps[id]; ok {
				items = append(items, linkedStepFromProto(step))
			}
		}
		return apiresource.NewList(items, apiresource.PageInfo{})
	}
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "in_steps", linked(s.InStepIds))
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "out_steps", linked(s.OutStepIds))

	// machines is expandable: stash the ids; LoadMachines fetches the real machines. Never fabricate.
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "machine_ids", s.MachineIds)

	if s.DepartmentId != nil && *s.DepartmentId != "" {
		meta.Set(constants.ObjectTypeProductionStep, s.Id, "department_id", *s.DepartmentId)
	}

	if s.ScanningStationId != nil && *s.ScanningStationId != "" {
		meta.Set(constants.ObjectTypeProductionStep, s.Id, "scanning_station_id", *s.ScanningStationId)
	}

	var notes *string
	if s.Notes != nil && *s.Notes != "" {
		notes = s.Notes
	}

	return apiresource.ProductionFlowStep{
		ID:             s.Id,
		Object:         constants.ObjectTypeProductionStep,
		Name:           s.Name,
		Notes:          notes,
		LevelingFactor: s.LevelingFactor,
		Allowances:     s.Allowances,
		LaborRate:      resourceloaders.UnitPriceFromProto(s.LaborRate),
		LaborTime:      resourceloaders.RateFromProto(s.LaborTime),
		OverheadRate:   resourceloaders.UnitPriceFromProto(s.OverheadRate),
		CreatedAt:      grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(s.UpdatedAt),
	}
}

func ProductionFlowPresenter(ctx context.Context, steps []*pb.ProductionFlowStepInfo) (*apiresource.ProductionFlow, *apierror.APIError) {
	byID := make(map[string]*pb.ProductionFlowStepInfo, len(steps))
	for _, s := range steps {
		byID[s.Id] = s
	}
	flowSteps := make([]apiresource.ProductionFlowStep, 0, len(steps))
	for _, s := range steps {
		flowSteps = append(flowSteps, productionFlowStepFromProto(ctx, s, byID))
	}

	meta := include.GetLoadMeta(ctx)
	if apiErr := fillFlowUnits(ctx, meta, flowSteps); apiErr != nil {
		return nil, apiErr
	}
	meta.Set(constants.ObjectTypeProductionFlow, "singleton", "steps",
		apiresource.NewList(flowSteps, apiresource.PageInfo{}))

	return &apiresource.ProductionFlow{
		Object: constants.ObjectTypeProductionFlow,
	}, nil
}

// fillFlowUnits loads the units of the values each step stashes for its includes: they are attached after the
// response leaves the service, so they are filled here rather than with the response.
func fillFlowUnits(ctx context.Context, meta *include.LoadMeta, steps []apiresource.ProductionFlowStep) *apierror.APIError {
	values := []any{steps}
	for _, s := range steps {
		for _, key := range []string{"production", "consumptions", "in_steps", "out_steps"} {
			if v, ok := meta.Get(constants.ObjectTypeProductionStep, s.ID, key); ok {
				values = append(values, v)
			}
		}
	}
	return resourceloaders.FillUnits(ctx, values...)
}
