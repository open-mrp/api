package grpc

import (
	"sort"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// productionCostScale is the decimal places a production cost figure is reported to. Steps are costed through base-unit ratios stored to 30 places and divided to 16, so digits past this are rounding, not cost.
const productionCostScale = 10

func productionCostReportToProto(r *domain.ProductionCostReport) *pb.AnalyzeProductionCostsResponse {
	unitIDs := make([]string, 0, len(r.Units))
	for id := range r.Units {
		unitIDs = append(unitIDs, id)
	}
	sort.Strings(unitIDs)
	units := make([]*pb.UnitInfo, 0, len(unitIDs))
	for _, id := range unitIDs {
		if u := r.Units[id]; u != nil {
			units = append(units, unitToProto(u))
		}
	}
	return &pb.AnalyzeProductionCostsResponse{
		CurrencyUnitId:       r.CurrencyUnitID,
		TimeUnitId:           r.TimeUnitID,
		Units:                units,
		Totals:               productionCostSetToProto(r.Totals),
		Departments:          productionCostGroupsToProto(r.Departments),
		Categories:           productionCostGroupsToProto(r.Categories),
		DepartmentCategories: productionCostGroupsToProto(r.DepartmentCategories),
	}
}

func productionCostGroupsToProto(groups []domain.ProductionCostGroup) []*pb.ProductionCostGroupProto {
	out := make([]*pb.ProductionCostGroupProto, len(groups))
	for i, g := range groups {
		out[i] = &pb.ProductionCostGroupProto{
			Department: productionCostRefToProto(g.Department),
			Category:   productionCostRefToProto(g.Category),
			Costs:      productionCostSetToProto(g.Costs),
		}
	}
	return out
}

func productionCostRefToProto(ref *domain.ProductionCostRef) *pb.BasicInfoProto {
	if ref == nil {
		return nil
	}
	return &pb.BasicInfoProto{Id: ref.ID, Name: ref.Name}
}

func productionCostSetToProto(s domain.ProductionCostSet) *pb.ProductionCostSetProto {
	return &pb.ProductionCostSetProto{
		Productive: productionCostToProto(s.Productive),
		Seconds:    productionCostToProto(s.Seconds),
		Waste:      productionCostToProto(s.Waste),
		Total:      productionCostToProto(s.Total),
	}
}

func productionCostToProto(c domain.ProductionCost) *pb.ProductionCostProto {
	produced := make([]*pb.ProductionCostProducedProto, len(c.Produced))
	for i, p := range c.Produced {
		produced[i] = &pb.ProductionCostProducedProto{UnitId: p.UnitID, Value: productionCostFigure(p.Value)}
	}
	return &pb.ProductionCostProto{
		Materials: productionCostFigure(c.Materials),
		Labor:     productionCostFigure(c.Labor),
		Overhead:  productionCostFigure(c.Overhead),
		Total:     productionCostFigure(c.Total),
		LaborTime: productionCostFigure(c.LaborHours),
		Produced:  produced,
	}
}

func productionCostFigure(d decimal.Decimal) string {
	return d.Round(productionCostScale).String()
}
