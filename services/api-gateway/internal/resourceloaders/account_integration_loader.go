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

var accountIntegrationLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.account_integration")

// LoadAccountIntegrations fetches account integrations by ID via
// BatchGetAccountIntegrationsByIDs. Pure leaf — no expandable sub-resources.
func LoadAccountIntegrations(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountIntegrationLoaderTracer, "loader.account_integrations.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAccountIntegrationsByIDsResponse, error) {
			return coreClient.BatchGetAccountIntegrationsByIDs(ctx, &pb.BatchGetAccountIntegrationsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.AccountIntegrations))
	for _, ai := range resp.AccountIntegrations {
		out[ai.Id] = AccountIntegrationFromProto(ai)
	}
	return out, nil
}

// AccountIntegrationFromProto maps the gRPC AccountIntegrationInfo to the
// apiresource shape. Exported so endpoint service methods that already hold a
// proto response (e.g. Delete, which can't fan back through LoadXs) can
// reuse it directly.
func AccountIntegrationFromProto(ai *pb.AccountIntegrationInfo) *apiresource.AccountIntegration {
	return &apiresource.AccountIntegration{
		ID:              ai.Id,
		Object:          constants.ObjectTypeAccountIntegration,
		Name:            ai.Name,
		IntegrationCode: constants.IntegrationCode(ai.IntegrationCode),
		Status:          constants.AccountIntegrationStatusFromActive(ai.IsActive),
		CreatedAt:       grpcutil.TimestampToTime(ai.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(ai.UpdatedAt),
	}
}
