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

var accountGroupProductLineAccessLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.account_group_product_line_access")

// LoadAccountGroupProductLineAccess fetches access records by account_group_id via BatchGetAccountGroupProductLineAccessByIDs. The inline AccountGroup shell + ProductLines list are built from denormalized proto fields — no expandable sub-resources.
func LoadAccountGroupProductLineAccess(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupProductLineAccessLoaderTracer, "loader.account_group_product_line_access.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAccountGroupProductLineAccessByIDsResponse, error) {
			return coreClient.BatchGetAccountGroupProductLineAccessByIDs(ctx, &pb.BatchGetAccountGroupProductLineAccessByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Items))
	for _, item := range resp.Items {
		out[item.AccountGroupId] = AccountGroupProductLineAccessFromProto(item)
	}
	return out, nil
}

// AccountGroupProductLineAccessFromProto maps the gRPC proto to the apiresource. Exported so endpoint service methods that already hold a proto response (Create/Update return the resource directly) can reuse it.
func AccountGroupProductLineAccessFromProto(item *pb.AccountGroupProductLineAccessInfo) *apiresource.AccountGroupProductLineAccess {
	productLines := make([]apiresource.ProductLine, len(item.ProductLines))
	for i, pl := range item.ProductLines {
		productLines[i] = apiresource.ProductLine{
			ID:     pl.Id,
			Object: constants.ObjectTypeProductLine,
			Name:   pl.Name,
		}
	}
	return &apiresource.AccountGroupProductLineAccess{
		AccountGroup: &apiresource.AccountGroup{
			ID:     item.AccountGroupId,
			Object: constants.ObjectTypeAccountGroup,
			Name:   item.AccountGroupName,
		},
		Object:       constants.ObjectTypeAccountGroupProductLineAccess,
		ProductLines: apiresource.NewList(productLines, apiresource.PageInfo{}),
		CreatedAt:    grpcutil.TimestampToTime(item.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(item.UpdatedAt),
	}
}
