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

var accountStatusLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.account_status")

// LoadAccountStatuses fetches account statuses by ID via
// BatchGetAccountStatusesByIDs. System-only resource — no LoadMeta needed.
func LoadAccountStatuses(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountStatusLoaderTracer, "loader.account_statuses.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAccountStatusesByIDsResponse, error) {
			return coreClient.BatchGetAccountStatusesByIDs(ctx, &pb.BatchGetAccountStatusesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.AccountStatuses))
	for _, as := range resp.AccountStatuses {
		out[as.Id] = accountStatusFromProto(as)
	}
	return out, nil
}

func accountStatusFromProto(as *pb.AccountStatusInfo) *apiresource.AccountStatus {
	return &apiresource.AccountStatus{
		ID:        as.Id,
		Object:    constants.ObjectTypeAccountStatus,
		Code:      constants.AccountStatusCode(as.Code),
		Name:      as.Name,
		CreatedAt: grpcutil.TimestampToTime(as.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(as.UpdatedAt),
	}
}
