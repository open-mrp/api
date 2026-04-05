package productionflowep

import (
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
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
		Unit:         nil, // Populated via include expansion
	}
}

func flowRatePresenter(r *pb.RateInfo) *apiresource.Rate {
	if r == nil {
		return nil
	}
	return &apiresource.Rate{
		ID:     r.Id,
		Object: constants.ObjectTypeRate,
		Value:  r.Value,
		NumeratorUnit: &apiresource.Unit{
			ID:     r.NumeratorUnitId,
			Object: constants.ObjectTypeUnit,
		},
		DenominatorUnit: &apiresource.Unit{
			ID:     r.DenominatorUnitId,
			Object: constants.ObjectTypeUnit,
		},
		DisplayValue: apiresource.FormatRateDisplayValue(r.Value, r.NumeratorUnitAbbreviation, r.NumeratorUnitType, r.DenominatorUnitAbbreviation),
	}
}

func flowProductionPresenter(p *pb.ProductionFlowProductionInfo) *apiresource.ProductionFlowProduction {
	if p == nil {
		return nil
	}
	var item *apiresource.ProductionFlowItemRef
	if p.ItemId != "" {
		item = &apiresource.ProductionFlowItemRef{
			ID:     p.ItemId,
			Object: constants.ObjectTypeItem,
			SKU:    p.ItemSku,
		}
	}
	return &apiresource.ProductionFlowProduction{
		ID:       p.Id,
		Object:   constants.ObjectTypeProduction,
		Item:     item,
		Quantity: flowQuantityPresenter(p.Quantity),
	}
}

func flowConsumptionPresenter(c *pb.ProductionFlowConsumptionInfo) apiresource.ProductionFlowConsumption {
	if c == nil {
		return apiresource.ProductionFlowConsumption{}
	}
	var item *apiresource.ProductionFlowItemRef
	if c.ItemId != "" {
		item = &apiresource.ProductionFlowItemRef{
			ID:     c.ItemId,
			Object: constants.ObjectTypeItem,
			SKU:    c.ItemSku,
		}
	}
	var instructions *string
	if c.Instructions != nil {
		instructions = c.Instructions
	}
	return apiresource.ProductionFlowConsumption{
		ID:            c.Id,
		Object:        constants.ObjectTypeConsumption,
		Item:          item,
		Quantity:      flowQuantityPresenter(c.Quantity),
		WasteQuantity: flowQuantityPresenter(c.WasteQuantity),
		Instructions:  instructions,
	}
}

func ProductionFlowStepPresenter(s *pb.ProductionFlowStepInfo) apiresource.ProductionFlowStep {
	if s == nil {
		return apiresource.ProductionFlowStep{}
	}

	consumptions := make([]apiresource.ProductionFlowConsumption, 0, len(s.Consumptions))
	for _, c := range s.Consumptions {
		consumptions = append(consumptions, flowConsumptionPresenter(c))
	}

	inSteps := make([]apiresource.ProductionFlowStepRef, 0, len(s.InStepIds))
	for _, id := range s.InStepIds {
		inSteps = append(inSteps, apiresource.ProductionFlowStepRef{
			ID:     id,
			Object: constants.ObjectTypeProductionStep,
		})
	}

	outSteps := make([]apiresource.ProductionFlowStepRef, 0, len(s.OutStepIds))
	for _, id := range s.OutStepIds {
		outSteps = append(outSteps, apiresource.ProductionFlowStepRef{
			ID:     id,
			Object: constants.ObjectTypeProductionStep,
		})
	}

	var scanningStation *apiresource.ProductionFlowStepRef
	if s.ScanningStationId != nil {
		scanningStation = &apiresource.ProductionFlowStepRef{
			ID:     *s.ScanningStationId,
			Object: constants.ObjectTypeScanningStation,
		}
	}

	return apiresource.ProductionFlowStep{
		ID:              s.Id,
		Object:          constants.ObjectTypeProductionStep,
		Name:            s.Name,
		Production:      flowProductionPresenter(s.Production),
		Consumptions:    consumptions,
		InSteps:         inSteps,
		OutSteps:        outSteps,
		ScanningStation: scanningStation,
		LevelingFactor:  s.LevelingFactor,
		Allowances:      s.Allowances,
		LaborRate:       flowRatePresenter(s.LaborRate),
		LaborTime:       flowRatePresenter(s.LaborTime),
		OverheadRate:    flowRatePresenter(s.OverheadRate),
	}
}

func ProductionFlowPresenter(steps []*pb.ProductionFlowStepInfo) *apiresource.ProductionFlow {
	flowSteps := make([]apiresource.ProductionFlowStep, 0, len(steps))
	for _, s := range steps {
		flowSteps = append(flowSteps, ProductionFlowStepPresenter(s))
	}
	return &apiresource.ProductionFlow{
		Object: constants.ObjectTypeProductionFlow,
		Steps:  flowSteps,
	}
}
