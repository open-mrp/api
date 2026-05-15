package productionstepep

import (
	"time"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

var embeddedRateTimestamp = time.Unix(1, 0).UTC()

func ratePresenter(r *pb.ProductionStepRateInfo) *apiresource.Rate {
	if r == nil {
		return nil
	}
	return &apiresource.Rate{
		ID:     r.Id,
		Object: constants.ObjectTypeRate,
		Value:  r.Value,
		NumeratorUnit: apiresource.ExpandableUnitStub(
			r.NumeratorUnitId,
			r.NumeratorUnitAbbreviation,
			r.NumeratorUnitAbbreviation,
			r.NumeratorUnitType,
			embeddedRateTimestamp,
		),
		DenominatorUnit: apiresource.ExpandableUnitStub(
			r.DenominatorUnitId,
			r.DenominatorUnitAbbreviation,
			r.DenominatorUnitAbbreviation,
			r.DenominatorUnitType,
			embeddedRateTimestamp,
		),
		DisplayValue: apiresource.FormatRateDisplayValue(r.Value, r.NumeratorUnitAbbreviation, r.NumeratorUnitType, r.DenominatorUnitAbbreviation),
		CreatedAt:    embeddedRateTimestamp,
		UpdatedAt:    embeddedRateTimestamp,
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

	var producedItem *apiresource.Item
	if p.ItemId != "" {
		itemTS := grpcutil.TimestampToTime(p.CreatedAt)
		producedItem = &apiresource.Item{
			ID:           p.ItemId,
			Object:       constants.ObjectTypeItem,
			SKU:          p.ItemSku,
			Description:  p.ItemDescription,
			ItemTypeCode: constants.ItemTypeCode(p.ItemTypeCode),
			CreatedAt:    itemTS,
			UpdatedAt:    itemTS,
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

func lightProductionStepToStub(st *pb.LightProductionStepInfo, fallback time.Time) apiresource.ProductionStep {
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

func ProductionStepPresenter(s *pb.ProductionStepInfo) apiresource.ProductionStep {
	if s == nil {
		return apiresource.ProductionStep{}
	}

	stepTS := grpcutil.TimestampToTime(s.CreatedAt)

	// Consumptions
	consumptions := make([]apiresource.Consumption, len(s.Consumptions))
	for i, c := range s.Consumptions {
		consumptions[i] = consumptionPresenter(c)
	}

	// Machines
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

	// Scanning station
	var scanStation *apiresource.ScanningStation
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
		scanStation = &apiresource.ScanningStation{
			ID:                  ss.Id,
			Object:              constants.ObjectTypeScanningStation,
			Name:                ss.Name,
			Type:                ssType,
			OperatorRequirement: constants.OperatorRequirementNone,
			CreatedAt:           ssCreated,
			UpdatedAt:           ssUpdated,
		}
	}

	// In/Out steps
	inSteps := make([]apiresource.ProductionStep, len(s.InSteps))
	for i, st := range s.InSteps {
		inSteps[i] = lightProductionStepToStub(st, stepTS)
	}

	outSteps := make([]apiresource.ProductionStep, len(s.OutSteps))
	for i, st := range s.OutSteps {
		outSteps[i] = lightProductionStepToStub(st, stepTS)
	}

	// Department
	var department *apiresource.Department
	if s.DepartmentId != nil && *s.DepartmentId != "" {
		department = &apiresource.Department{
			ID:        *s.DepartmentId,
			Object:    constants.ObjectTypeDepartment,
			Name:      "Department",
			CreatedAt: stepTS,
			UpdatedAt: stepTS,
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
		Consumptions:    apiresource.NewList(consumptions, apiresource.PageInfo{}),
		Machines:        apiresource.NewList(machines, apiresource.PageInfo{}),
		ScanningStation: scanStation,
		InSteps:         apiresource.NewList(inSteps, apiresource.PageInfo{}),
		OutSteps:        apiresource.NewList(outSteps, apiresource.PageInfo{}),
		Department:      department,
		CreatedAt:       stepTS,
		UpdatedAt:       grpcutil.TimestampToTime(s.UpdatedAt),
	}
}

func consumptionPresenter(c *pb.ConsumptionInfo) apiresource.Consumption {
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
		Quantity:      quantityPresenter(c.Quantity),
		WasteQuantity: quantityPresenter(c.WasteQuantity),
		ConsumedItem:  consumedItem,
		Instructions:  c.Instructions,
		CreatedAt:     itemTS,
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
