package analyticsep

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func (m *analyticsSvcImpl) AnalyzeProductionCosts(ctx context.Context, req *AnalyzeProductionCostsRequest) (*apiresource.AnalyzeProductionCostsResponse, *apierror.APIError) {
	pbReq := &pb.AnalyzeProductionCostsRequest{
		StartDate:      timestamppb.New(req.StartDate),
		EndDate:        timestamppb.New(req.EndDate),
		ItemIds:        req.ItemIDs,
		ProductLineIds: req.ProductLineIDs,
		DepartmentIds:  req.DepartmentIDs,
		CategoryIds:    req.CategoryIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, analyticsSvcTracer, "service.analytics.analyze_production_costs", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AnalyzeProductionCostsResponse, error) {
			return m.coreClient.AnalyzeProductionCosts(ctx, pbReq, opts...)
		}, grpcutil.WithTimeout(grpcutil.AnalyticsOperationTimeout))
	if apiErr != nil {
		return nil, apiErr
	}

	return presentProductionCosts(req, resp), nil
}

// presentProductionCosts renders the report against the units core attached to it.
func presentProductionCosts(req *AnalyzeProductionCostsRequest, resp *pb.AnalyzeProductionCostsResponse) *apiresource.AnalyzeProductionCostsResponse {
	units := make(map[string]*apiresource.Unit, len(resp.GetUnits()))
	for _, u := range resp.GetUnits() {
		units[u.GetId()] = reportUnit(u)
	}
	cost := func(c *pb.ProductionCostProto) *apiresource.ProductionCost {
		produced := make([]apiresource.ComputedQuantity, len(c.GetProduced()))
		for i, p := range c.GetProduced() {
			produced[i] = *openOrderQuantity(p.GetValue(), units[p.GetUnitId()])
		}
		return &apiresource.ProductionCost{
			Object:    constants.ObjectTypeProductionCost,
			Materials: c.GetMaterials(),
			Labor:     c.GetLabor(),
			Overhead:  c.GetOverhead(),
			Total:     c.GetTotal(),
			LaborTime: c.GetLaborTime(),
			Produced:  apiresource.NewList(produced, apiresource.PageInfo{}),
		}
	}

	departments := make([]apiresource.ProductionCostDepartment, len(resp.GetDepartments()))
	for i, g := range resp.GetDepartments() {
		costs := g.GetCosts()
		departments[i] = apiresource.ProductionCostDepartment{
			Object:     constants.ObjectTypeProductionCostDepartment,
			Department: productionCostEntity(g.GetDepartment(), constants.ObjectTypeDepartment),
			Productive: cost(costs.GetProductive()),
			Seconds:    cost(costs.GetSeconds()),
			Waste:      cost(costs.GetWaste()),
			Total:      cost(costs.GetTotal()),
		}
	}
	categories := make([]apiresource.ProductionCostCategory, len(resp.GetCategories()))
	for i, g := range resp.GetCategories() {
		costs := g.GetCosts()
		categories[i] = apiresource.ProductionCostCategory{
			Object:     constants.ObjectTypeProductionCostCategory,
			Category:   productionCostEntity(g.GetCategory(), constants.ObjectTypeItemCategory),
			Productive: cost(costs.GetProductive()),
			Seconds:    cost(costs.GetSeconds()),
			Waste:      cost(costs.GetWaste()),
			Total:      cost(costs.GetTotal()),
		}
	}
	departmentCategories := make([]apiresource.ProductionCostDepartmentCategory, len(resp.GetDepartmentCategories()))
	for i, g := range resp.GetDepartmentCategories() {
		costs := g.GetCosts()
		departmentCategories[i] = apiresource.ProductionCostDepartmentCategory{
			Object:     constants.ObjectTypeProductionCostDepartmentCategory,
			Department: productionCostEntity(g.GetDepartment(), constants.ObjectTypeDepartment),
			Category:   productionCostEntity(g.GetCategory(), constants.ObjectTypeItemCategory),
			Productive: cost(costs.GetProductive()),
			Seconds:    cost(costs.GetSeconds()),
			Waste:      cost(costs.GetWaste()),
			Total:      cost(costs.GetTotal()),
		}
	}

	totals := resp.GetTotals()
	return &apiresource.AnalyzeProductionCostsResponse{
		Object:       constants.ObjectTypeAnalyzeProductionCostsResponse,
		StartsAt:     req.StartDate,
		EndsAt:       req.EndDate,
		CurrencyUnit: units[resp.GetCurrencyUnitId()],
		TimeUnit:     units[resp.GetTimeUnitId()],
		Totals: &apiresource.ProductionCostTotals{
			Object:     constants.ObjectTypeProductionCostTotals,
			Productive: cost(totals.GetProductive()),
			Seconds:    cost(totals.GetSeconds()),
			Waste:      cost(totals.GetWaste()),
			Total:      cost(totals.GetTotal()),
		},
		Departments:          apiresource.NewList(departments, apiresource.PageInfo{}),
		Categories:           apiresource.NewList(categories, apiresource.PageInfo{}),
		DepartmentCategories: apiresource.NewList(departmentCategories, apiresource.PageInfo{}),
	}
}

func productionCostEntity(ref *pb.BasicInfoProto, objectType constants.ObjectType) *apiresource.Entity {
	if ref == nil {
		return nil
	}
	name := ref.GetName()
	return apiresource.NewEntity(ref.GetId(), objectType, &name, nil)
}
