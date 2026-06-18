package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var customerProductLineAccessLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.customer_product_line_access")

// LoadCustomerProductLineAccess fetches access records by customer_id via BatchGetCustomerProductLineAccessByIDs. Inline Customer shell and inline ProductLines list — no expandable sub-resources.
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
	out := make(map[string]any, len(resp.Items))
	for _, item := range resp.Items {
		out[item.CustomerId] = CustomerProductLineAccessFromProto(item)
	}
	return out, nil
}

// CustomerProductLineAccessFromProto maps the gRPC proto to the apiresource. Exported for use by mutation handlers that already hold a proto response.
// NOTE: preserves the legacy presenter's quirk of setting Customer.Object to ObjectTypeAccount rather than ObjectTypeCustomer (the apiresource.Customer type is account-shaped in the existing schema).
func CustomerProductLineAccessFromProto(item *pb.CustomerProductLineAccessInfo) *apiresource.CustomerProductLineAccess {
	productLines := make([]apiresource.ProductLine, len(item.ProductLines))
	for i, pl := range item.ProductLines {
		productLines[i] = apiresource.ProductLine{
			ID:     pl.Id,
			Object: constants.ObjectTypeProductLine,
			Name:   pl.Name,
		}
	}
	return &apiresource.CustomerProductLineAccess{
		Customer: &apiresource.Customer{
			ID:     item.CustomerId,
			Object: constants.ObjectTypeAccount,
			Name:   item.CustomerName,
			Number: item.CustomerNumber,
		},
		Object:       constants.ObjectTypeCustomerProductLineAccess,
		ProductLines: apiresource.NewList(productLines, apiresource.PageInfo{}),
		CreatedAt:    grpcutil.TimestampToTime(item.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(item.UpdatedAt),
	}
}
