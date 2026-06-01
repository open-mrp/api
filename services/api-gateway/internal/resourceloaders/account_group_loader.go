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

var accountGroupLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.account_group")

// LoadAccountGroups fetches account groups by ID via
// BatchGetAccountGroupsByIDs. AccountGroup exposes no expandable sub-resources
// (not even an Owner) so no LoadMeta needs to be stashed.
func LoadAccountGroups(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupLoaderTracer, "loader.account_groups.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAccountGroupsByIDsResponse, error) {
			return coreClient.BatchGetAccountGroupsByIDs(ctx, &pb.BatchGetAccountGroupsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.AccountGroups))
	for _, ag := range resp.AccountGroups {
		out[ag.Id] = accountGroupFromProto(ag)
	}
	return out, nil
}

func accountGroupFromProto(ag *pb.AccountGroupInfo) *apiresource.AccountGroup {
	return &apiresource.AccountGroup{
		ID:               ag.Id,
		Object:           constants.ObjectTypeAccountGroup,
		Name:             ag.Name,
		Description:      ag.Description,
		CommissionPolicy: constants.CommissionPolicy(ag.CommissionPolicy),
		FreightPolicy:    constants.FreightPolicy(ag.FreightPolicy),
		Type:             constants.AccountGroupType(ag.Type),
		CreatedAt:        grpcutil.TimestampToTime(ag.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(ag.UpdatedAt),
	}
}
