package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var customerProductLineAccessLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.customer_product_line_access")

// LoadCustomerProductLineAccess fetches access records by customer_id via BatchGetCustomerProductLineAccessByIDs and embeds the real customer and product line records, which whoever may read the access may read.
func LoadCustomerProductLineAccess(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, customerProductLineAccessLoaderTracer, "loader.customer_product_line_access.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetCustomerProductLineAccessByIDsResponse, error) {
			return coreClient.BatchGetCustomerProductLineAccessByIDs(ctx, &pb.BatchGetCustomerProductLineAccessByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(resp.Items) == 0 {
		return map[string]any{}, nil
	}

	customerIDs := make([]string, len(resp.Items))
	granted := make([][]*pb.ProductLineAccessInfo, len(resp.Items))
	for i, item := range resp.Items {
		customerIDs[i] = item.CustomerId
		granted[i] = item.ProductLines
	}
	embedCtx := resourcekit.WithIncludeReads(ctx)
	customers, apiErr := LoadCustomers(embedCtx, customerIDs)
	if apiErr != nil {
		return nil, apiErr
	}
	lines, apiErr := LoadProductLines(embedCtx, grantedProductLineIDs(granted...))
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]any, len(resp.Items))
	for _, item := range resp.Items {
		access := &apiresource.CustomerProductLineAccess{
			Object:       constants.ObjectTypeCustomerProductLineAccess,
			ProductLines: grantedProductLines(item.ProductLines, lines),
			CreatedAt:    grpcutil.TimestampToTime(item.CreatedAt),
			UpdatedAt:    grpcutil.TimestampToTime(item.UpdatedAt),
		}
		if customer, ok := customers[item.CustomerId].(*apiresource.Customer); ok {
			access.Customer = customer
		}
		out[item.CustomerId] = access
	}
	return out, nil
}

// grantedProductLineIDs is every product line the access records grant, each once.
func grantedProductLineIDs(granted ...[]*pb.ProductLineAccessInfo) []string {
	seen := map[string]bool{}
	var ids []string
	for _, lines := range granted {
		for _, pl := range lines {
			if !seen[pl.Id] {
				seen[pl.Id] = true
				ids = append(ids, pl.Id)
			}
		}
	}
	return ids
}

// grantedProductLines is the granted lines as their real records.
func grantedProductLines(granted []*pb.ProductLineAccessInfo, loaded map[string]any) *apiresource.List[apiresource.ProductLine] {
	items := make([]apiresource.ProductLine, 0, len(granted))
	for _, pl := range granted {
		if line, ok := loaded[pl.Id].(*apiresource.ProductLine); ok {
			items = append(items, *line)
		}
	}
	return apiresource.NewList(items, apiresource.PageInfo{})
}
