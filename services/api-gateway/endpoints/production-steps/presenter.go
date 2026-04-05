package productionstepep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ratePresenter(r *pb.ProductionStepRateInfo) *apiresource.Rate {
	if r == nil {
		return nil
	}
	return &apiresource.Rate{
		ID:     r.Id,
		Object: constants.ObjectTypeRate,
		Value:  r.Value,
		NumeratorUnit: &apiresource.Unit{
			ID:           r.NumeratorUnitId,
			Object:       constants.ObjectTypeUnit,
			Abbreviation: r.NumeratorUnitAbbreviation,
			Type:         constants.UnitType(r.NumeratorUnitType),
		},
		DenominatorUnit: &apiresource.Unit{
			ID:           r.DenominatorUnitId,
			Object:       constants.ObjectTypeUnit,
			Abbreviation: r.DenominatorUnitAbbreviation,
			Type:         constants.UnitType(r.DenominatorUnitType),
		},
		DisplayValue: apiresource.FormatRateDisplayValue(r.Value, r.NumeratorUnitAbbreviation, r.NumeratorUnitType, r.DenominatorUnitAbbreviation),
	}
}

func quantityPresenter(q *pb.QuantityInfo) *apiresource.Quantity {
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

func productionOutputPresenter(p *pb.ProductionInfo) *apiresource.ProductionOutput {
	if p == nil {
		return nil
	}

	var producedItem *apiresource.ConsumptionItem
	if p.ItemId != "" {
		producedItem = &apiresource.ConsumptionItem{
			ID:           p.ItemId,
			Object:       constants.ObjectTypeItem,
			SKU:          p.ItemSku,
			Description:  p.ItemDescription,
			ItemTypeCode: constants.ItemTypeCode(p.ItemTypeCode),
		}
	}

	return &apiresource.ProductionOutput{
		ID:           p.Id,
		Object:       constants.ObjectTypeProduction,
		ProducedItem: producedItem,
		Quantity:     quantityPresenter(p.Quantity),
		CreatedAt:    grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(p.UpdatedAt),
	}
}

func ProductionStepPresenter(s *pb.ProductionStepInfo) apiresource.ProductionStep {
	if s == nil {
		return apiresource.ProductionStep{}
	}

	// Consumptions
	consumptions := make([]apiresource.Consumption, len(s.Consumptions))
	for i, c := range s.Consumptions {
		consumptions[i] = consumptionPresenter(c)
	}

	// Machines
	machines := make([]apiresource.ProductionStepMachine, len(s.Machines))
	for i, m := range s.Machines {
		machines[i] = apiresource.ProductionStepMachine{
			ID:     m.Id,
			Object: constants.ObjectTypeMachine,
			Name:   m.Name,
		}
	}

	// Scanning station
	var scanStation *apiresource.ProductionStepScanStation
	if s.ScanningStation != nil {
		scanStation = &apiresource.ProductionStepScanStation{
			ID:     s.ScanningStation.Id,
			Object: constants.ObjectTypeScanningStation,
			Name:   s.ScanningStation.Name,
		}
	}

	// In/Out steps
	inSteps := make([]apiresource.ProductionStepRef, len(s.InSteps))
	for i, st := range s.InSteps {
		inSteps[i] = apiresource.ProductionStepRef{
			ID:     st.Id,
			Object: constants.ObjectTypeProductionStep,
			Name:   st.Name,
		}
	}

	outSteps := make([]apiresource.ProductionStepRef, len(s.OutSteps))
	for i, st := range s.OutSteps {
		outSteps[i] = apiresource.ProductionStepRef{
			ID:     st.Id,
			Object: constants.ObjectTypeProductionStep,
			Name:   st.Name,
		}
	}

	// Department
	var department *apiresource.ProductionStepDepartment
	if s.DepartmentId != nil && *s.DepartmentId != "" {
		department = &apiresource.ProductionStepDepartment{
			ID:     *s.DepartmentId,
			Object: constants.ObjectTypeDepartment,
		}
	}

	return apiresource.ProductionStep{
		ID:              s.Id,
		Object:          constants.ObjectTypeProductionStep,
		Name:            s.Name,
		Notes:           s.Notes,
		LevelingFactor:  s.LevelingFactor,
		Allowances:      s.Allowances,
		LaborRate:       ratePresenter(s.LaborRate),
		LaborTime:       ratePresenter(s.LaborTime),
		OverheadRate:    ratePresenter(s.OverheadRate),
		Production:      productionOutputPresenter(s.Production),
		Consumptions:    consumptions,
		Machines:        machines,
		ScanningStation: scanStation,
		InSteps:         inSteps,
		OutSteps:        outSteps,
		Department:      department,
		CreatedAt:       grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(s.UpdatedAt),
	}
}

func consumptionPresenter(c *pb.ConsumptionInfo) apiresource.Consumption {
	if c == nil {
		return apiresource.Consumption{}
	}

	var consumedItem *apiresource.ConsumptionItem
	if c.ItemId != "" {
		consumedItem = &apiresource.ConsumptionItem{
			ID:           c.ItemId,
			Object:       constants.ObjectTypeItem,
			SKU:          c.ItemSku,
			Description:  c.ItemDescription,
			ItemTypeCode: constants.ItemTypeCode(c.ItemTypeCode),
		}
	}

	return apiresource.Consumption{
		ID:            c.Id,
		Object:        constants.ObjectTypeConsumption,
		Quantity:      quantityPresenter(c.Quantity),
		WasteQuantity: quantityPresenter(c.WasteQuantity),
		ConsumedItem:  consumedItem,
		Instructions:  c.Instructions,
		CreatedAt:     grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:     grpcutil.TimestampToTime(c.UpdatedAt),
	}
}

func ProductionStepListPresenter(resp *pb.ListProductionStepsResponse) *apiresource.List[apiresource.ProductionStep] {
	steps := make([]apiresource.ProductionStep, len(resp.ProductionSteps))
	for i, s := range resp.ProductionSteps {
		steps[i] = ProductionStepPresenter(s)
	}
	return apiresource.NewList(steps, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
