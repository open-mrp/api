package owner

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var resolverTracer = tracing.GetTracer("api-gateway.internal.owner.resolver")

// ResolveOwnerAccount fetches full account details when the "owner.account"
// include is requested. Returns nil if the include is not requested, the
// account ID is nil/empty, or the gRPC call fails.
func ResolveOwnerAccount(ctx context.Context, coreClient pb.CoreServiceClient, accountID *string) *apiresource.Account {
	if accountID == nil || *accountID == "" {
		return nil
	}
	if !appctx.IsIncludeRequested(ctx, "owner.account") {
		return nil
	}

	resp, apiErr := grpcutil.CallRPC(ctx, resolverTracer, "owner.resolve_account", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountResponse, error) {
			return coreClient.GetAccount(ctx, &pb.GetAccountRequest{Id: *accountID}, opts...)
		})
	if apiErr != nil {
		return nil
	}

	a := resp.Account
	if a == nil {
		return nil
	}

	return &apiresource.Account{
		ID:        a.Id,
		Object:    constants.ObjectTypeAccount,
		Name:      a.Name,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
}
