package productionflowep

import (
	"context"
	"time"

	quantityep "github.com/open-mrp/api/services/api-gateway/endpoints/quantities"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func flowQuantityPresenter(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	norm := apiresource.NormalizeQuantityValue(q.Value, q.UnitType)
	return &apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        norm,
		DisplayValue: apiresource.FormatDisplayValue(norm, q.UnitAbbreviation, q.UnitType),
		Unit:         quantityep.UnitFromQuantityInfo(q),
	}
}

func flowRatePresenter(r *pb.RateInfo) *apiresource.Rate {
	if r == nil {
		return nil
	}
	return &apiresource.Rate{
		ID:              r.Id,
		Object:          constants.ObjectTypeRate,
		Value:           r.Value,
		NumeratorUnit:   unitFromRateInfo(r.NumeratorUnitId, r.NumeratorUnitName, r.NumeratorUnitAbbreviation, r.NumeratorUnitType, r.NumeratorUnitRatioNumerator, r.NumeratorUnitRatioDenominator, r.NumeratorUnitOffsetNumerator, r.NumeratorUnitOffsetDenominator, r.NumeratorUnitCreatedAt, r.NumeratorUnitUpdatedAt),
		DenominatorUnit: unitFromRateInfo(r.DenominatorUnitId, r.DenominatorUnitName, r.DenominatorUnitAbbreviation, r.DenominatorUnitType, r.DenominatorUnitRatioNumerator, r.DenominatorUnitRatioDenominator, r.DenominatorUnitOffsetNumerator, r.DenominatorUnitOffsetDenominator, r.DenominatorUnitCreatedAt, r.DenominatorUnitUpdatedAt),
		DisplayValue:    apiresource.FormatRateDisplayValue(r.Value, r.NumeratorUnitAbbreviation, r.NumeratorUnitType, r.DenominatorUnitAbbreviation),
		CreatedAt:       grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(r.UpdatedAt),
	}
}

func flowProductionPresenter(ctx context.Context, p *pb.ProductionFlowProductionInfo, ts time.Time) *apiresource.ProductionFlowProduction {
	if p == nil {
		return nil
	}
	if p.ItemId != "" {
		// produced_item is expandable: stash only the FK id; LoadItems fetches the
		// real Item on ?include=...produced_item. Never fabricate.
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeProduction, p.Id, "produced_item_id", p.ItemId)
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
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeConsumption, c.Id, "consumed_item_id", c.ItemId)
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

func productionFlowStepFromProto(ctx context.Context, s *pb.ProductionFlowStepInfo) apiresource.ProductionFlowStep {
	if s == nil {
		return apiresource.ProductionFlowStep{}
	}

	stubTS := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	meta := resourcekit.GetLoadMeta(ctx)

	production := flowProductionPresenter(ctx, s.Production, stubTS)
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "production", production)

	consumptionItems := make([]apiresource.ProductionFlowConsumption, 0, len(s.Consumptions))
	for _, c := range s.Consumptions {
		consumptionItems = append(consumptionItems, flowConsumptionPresenter(ctx, c, stubTS))
	}
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "consumptions",
		apiresource.NewList(consumptionItems, apiresource.PageInfo{}))

	inStepItems := make([]apiresource.ProductionStep, 0, len(s.InStepIds))
	for _, stepID := range s.InStepIds {
		inStepItems = append(inStepItems, apiresource.ProductionStep{
			ID:             stepID,
			Object:         constants.ObjectTypeProductionStep,
			Name:           "Production Step",
			LevelingFactor: "0",
			Allowances:     "0",
			CreatedAt:      stubTS,
			UpdatedAt:      stubTS,
		})
	}
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "in_steps",
		apiresource.NewList(inStepItems, apiresource.PageInfo{}))

	outStepItems := make([]apiresource.ProductionStep, 0, len(s.OutStepIds))
	for _, stepID := range s.OutStepIds {
		outStepItems = append(outStepItems, apiresource.ProductionStep{
			ID:             stepID,
			Object:         constants.ObjectTypeProductionStep,
			Name:           "Production Step",
			LevelingFactor: "0",
			Allowances:     "0",
			CreatedAt:      stubTS,
			UpdatedAt:      stubTS,
		})
	}
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "out_steps",
		apiresource.NewList(outStepItems, apiresource.PageInfo{}))

	machineItems := make([]apiresource.Machine, 0, len(s.MachineIds))
	for _, id := range s.MachineIds {
		machineItems = append(machineItems, apiresource.Machine{
			ID:           id,
			Object:       constants.ObjectTypeMachine,
			Name:         "Machine",
			SerialNumber: "—",
			CreatedAt:    stubTS,
			UpdatedAt:    stubTS,
		})
	}
	meta.Set(constants.ObjectTypeProductionStep, s.Id, "machines",
		apiresource.NewList(machineItems, apiresource.PageInfo{}))

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
		LaborRate:      flowRatePresenter(s.LaborRate),
		LaborTime:      flowRatePresenter(s.LaborTime),
		OverheadRate:   flowRatePresenter(s.OverheadRate),
		CreatedAt:      grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(s.UpdatedAt),
	}
}

func ProductionFlowPresenter(ctx context.Context, steps []*pb.ProductionFlowStepInfo) *apiresource.ProductionFlow {
	flowSteps := make([]apiresource.ProductionFlowStep, 0, len(steps))
	for _, s := range steps {
		flowSteps = append(flowSteps, productionFlowStepFromProto(ctx, s))
	}

	enrichFlowUnits(ctx, flowSteps)

	meta := resourcekit.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeProductionFlow, "singleton", "steps",
		apiresource.NewList(flowSteps, apiresource.PageInfo{}))

	return &apiresource.ProductionFlow{
		Object: constants.ObjectTypeProductionFlow,
	}
}

func enrichFlowUnits(ctx context.Context, steps []apiresource.ProductionFlowStep) {
	unitIDs := map[string]struct{}{}

	collectRateUnits := func(r *apiresource.Rate) {
		if r == nil {
			return
		}
		if r.NumeratorUnit != nil {
			unitIDs[r.NumeratorUnit.ID] = struct{}{}
		}
		if r.DenominatorUnit != nil {
			unitIDs[r.DenominatorUnit.ID] = struct{}{}
		}
	}
	collectQuantityUnit := func(q *apiresource.Quantity) {
		if q != nil && q.Unit != nil {
			unitIDs[q.Unit.ID] = struct{}{}
		}
	}

	meta := resourcekit.GetLoadMeta(ctx)
	for _, s := range steps {
		collectRateUnits(s.LaborRate)
		collectRateUnits(s.LaborTime)
		collectRateUnits(s.OverheadRate)

		if prod, ok := meta.Get(constants.ObjectTypeProductionStep, s.ID, "production"); ok {
			if p, ok := prod.(*apiresource.ProductionFlowProduction); ok && p != nil {
				collectQuantityUnit(p.Quantity)
			}
		}
		if consList, ok := meta.Get(constants.ObjectTypeProductionStep, s.ID, "consumptions"); ok {
			if list, ok := consList.(*apiresource.List[apiresource.ProductionFlowConsumption]); ok {
				for i := range list.Data {
					collectQuantityUnit(list.Data[i].Quantity)
					collectQuantityUnit(list.Data[i].WasteQuantity)
				}
			}
		}
	}

	if len(unitIDs) == 0 {
		return
	}

	ids := make([]string, 0, len(unitIDs))
	for id := range unitIDs {
		ids = append(ids, id)
	}

	loaded, apiErr := resourceloaders.LoadUnits(ctx, ids)
	if apiErr != nil || len(loaded) == 0 {
		return
	}

	replaceUnit := func(u *apiresource.Unit) {
		if u == nil {
			return
		}
		if v, ok := loaded[u.ID]; ok {
			if full, ok := v.(*apiresource.Unit); ok {
				*u = *full
			}
		}
	}
	replaceRateUnits := func(r *apiresource.Rate) {
		if r == nil {
			return
		}
		replaceUnit(r.NumeratorUnit)
		replaceUnit(r.DenominatorUnit)
		r.DisplayValue = apiresource.FormatRateDisplayValue(
			r.Value,
			r.NumeratorUnit.Abbreviation,
			string(r.NumeratorUnit.Type),
			r.DenominatorUnit.Abbreviation,
		)
	}
	replaceQuantityUnit := func(q *apiresource.Quantity) {
		if q == nil || q.Unit == nil {
			return
		}
		replaceUnit(q.Unit)
		q.DisplayValue = apiresource.FormatDisplayValue(q.Value, q.Unit.Abbreviation, string(q.Unit.Type))
	}

	for i := range steps {
		replaceRateUnits(steps[i].LaborRate)
		replaceRateUnits(steps[i].LaborTime)
		replaceRateUnits(steps[i].OverheadRate)

		if prod, ok := meta.Get(constants.ObjectTypeProductionStep, steps[i].ID, "production"); ok {
			if p, ok := prod.(*apiresource.ProductionFlowProduction); ok && p != nil {
				replaceQuantityUnit(p.Quantity)
			}
		}
		if consList, ok := meta.Get(constants.ObjectTypeProductionStep, steps[i].ID, "consumptions"); ok {
			if list, ok := consList.(*apiresource.List[apiresource.ProductionFlowConsumption]); ok {
				for j := range list.Data {
					replaceQuantityUnit(list.Data[j].Quantity)
					replaceQuantityUnit(list.Data[j].WasteQuantity)
				}
			}
		}
	}
}

// unitFromRateInfo builds a Unit from the real unit fields the production-flow
// proto carries inline. It never fabricates identifiers; ratio/offset default to
// the mathematical identity only when the proto omits them so the Unit stays valid.
func unitFromRateInfo(id, name, abbreviation, unitType, ratioNum, ratioDen, offsetNum, offsetDen string, createdAt, updatedAt *timestamppb.Timestamp) *apiresource.Unit {
	if ratioNum == "" {
		ratioNum = "1"
	}
	if ratioDen == "" {
		ratioDen = "1"
	}
	if offsetNum == "" {
		offsetNum = "0"
	}
	if offsetDen == "" {
		offsetDen = "1"
	}
	return &apiresource.Unit{
		ID:                id,
		Object:            constants.ObjectTypeUnit,
		Name:              name,
		Abbreviation:      abbreviation,
		Type:              constants.UnitType(unitType),
		RatioNumerator:    ratioNum,
		RatioDenominator:  ratioDen,
		OffsetNumerator:   offsetNum,
		OffsetDenominator: offsetDen,
		CreatedAt:         grpcutil.TimestampToTime(createdAt),
		UpdatedAt:         grpcutil.TimestampToTime(updatedAt),
	}
}
