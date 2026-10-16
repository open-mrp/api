package salesorderep

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// SalesOrderFromProto is an order as the API returns it, with what its includes load stashed beside it.
func SalesOrderFromProto(ctx context.Context, info *pb.SalesOrderInfo) apiresource.SalesOrder {
	so := resourceloaders.SalesOrderFromProto(ctx, info)
	meta := include.GetLoadMeta(ctx)

	// Totals: core computes them when the totals include is requested; an older core sends none, and the gateway
	// sums the lines itself. A line core could not price leaves them null rather than wrong.
	totals, handled := salesOrderTotalsFromProto(info)
	if !handled {
		totals = salesOrderTotalsFromOrder(info)
	}
	if totals != nil {
		meta.Set(constants.ObjectTypeSalesOrder, so.ID, "totals", totals)
	}

	lines := make([]apiresource.SalesOrderLine, len(info.Lines))
	for i, l := range info.Lines {
		lines[i] = salesOrderLineDetailFromProto(l)
		stashSalesOrderLineMeta(meta, l, &lines[i])
	}
	meta.Set(constants.ObjectTypeSalesOrder, so.ID, "lines", apiresource.NewList(lines, apiresource.PageInfo{}))
	return so
}
