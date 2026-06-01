package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var sandboxLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.sandbox")

// LoadSandboxes fetches sandboxes by ID via BatchGetSandboxesByIDs. Stashes
// owner_account_id in LoadMeta so the `owner_account` SubField closure can
// build the Account expansion when requested.
func LoadSandboxes(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, sandboxLoaderTracer, "loader.sandboxes.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetSandboxesByIDsResponse, error) {
			return coreClient.BatchGetSandboxesByIDs(ctx, &pb.BatchGetSandboxesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Sandboxes))
	for _, s := range resp.Sandboxes {
		out[s.Id] = sandboxFromProto(s)
		var ownerAccountID string
		if s.OwnerAccountId != nil {
			ownerAccountID = *s.OwnerAccountId
		}
		meta.Set(constants.ObjectTypeSandbox, s.Id, "owner_account_id", ownerAccountID)
	}
	return out, nil
}

func sandboxFromProto(s *pb.SandboxInfo) *apiresource.Sandbox {
	return &apiresource.Sandbox{
		ID:        s.Id,
		Object:    constants.ObjectTypeSandbox,
		Name:      s.Name,
		CreatedAt: grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(s.UpdatedAt),
	}
}
