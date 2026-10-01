package grpc

import (
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// ProductionRunBatchSummariesFromProto maps a run's per-item batch totals. Shared by the
// production-run endpoints and the include loader, which cannot import each other.
func ProductionRunBatchSummariesFromProto(in []*pb.ProductionRunBatchSummaryInfo) *apiresource.List[apiresource.ProductionRunBatchSummary] {
	out := make([]apiresource.ProductionRunBatchSummary, len(in))
	for i, s := range in {
		out[i] = apiresource.ProductionRunBatchSummary{
			Object:        constants.ObjectTypeProductionRunBatchSummary,
			Item:          apiresource.NewEntity(s.ItemId, constants.ObjectTypeItem, &s.ItemSku, nil),
			Unit:          apiresource.NewEntity(s.UnitId, constants.ObjectTypeUnit, &s.UnitAbbreviation, nil),
			QuantityValue: s.QuantityValue,
			BatchCount:    s.BatchCount,
		}
	}
	return apiresource.NewList(out, apiresource.PageInfo{})
}
