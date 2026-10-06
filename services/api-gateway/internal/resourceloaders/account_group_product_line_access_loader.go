package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var accountGroupProductLineAccessLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.account_group_product_line_access")

// LoadAccountGroupProductLineAccess fetches access records by account_group_id via BatchGetAccountGroupProductLineAccessByIDs and embeds the real account group and product line records, loaded as the caller.
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
	if len(resp.Items) == 0 {
		return map[string]any{}, nil
	}

	groupIDs := make([]string, len(resp.Items))
	granted := make([][]*pb.ProductLineAccessInfo, len(resp.Items))
	for i, item := range resp.Items {
		groupIDs[i] = item.AccountGroupId
		granted[i] = item.ProductLines
	}
	groups, _, apiErr := loadReadable(ctx, LoadAccountGroups, groupIDs)
	if apiErr != nil {
		return nil, apiErr
	}
	lines, linesReadable, apiErr := loadReadable(ctx, LoadProductLines, grantedProductLineIDs(granted...))
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]any, len(resp.Items))
	for _, item := range resp.Items {
		access := &apiresource.AccountGroupProductLineAccess{
			Object:       constants.ObjectTypeAccountGroupProductLineAccess,
			ProductLines: grantedProductLines(item.ProductLines, lines, linesReadable),
			CreatedAt:    grpcutil.TimestampToTime(item.CreatedAt),
			UpdatedAt:    grpcutil.TimestampToTime(item.UpdatedAt),
		}
		if group, ok := groups[item.AccountGroupId].(*apiresource.AccountGroup); ok {
			access.AccountGroup = group
		}
		out[item.AccountGroupId] = access
	}
	return out, nil
}
