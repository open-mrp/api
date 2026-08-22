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

var accountLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.account")

// LoadAccounts fetches accounts by ID via BatchGetAccountsByIDs. Builds clean *apiresource.Account values. No FK metadata is currently stashed — Account sub-resources (Branding, Portal, default addresses) aren't part of the carrier pilot.
func LoadAccounts(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountLoaderTracer, "loader.accounts.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetAccountsByIDsResponse, error) {
			return coreClient.BatchGetAccountsByIDs(ctx, &pb.BatchGetAccountsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Accounts))
	for _, a := range resp.Accounts {
		out[a.Id] = accountFromProto(a)
	}
	return out, nil
}

func LoadPublicAccounts(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadPublicAccounts should not be called — public accounts are not used as expandable sub-resources",
	)
}

func accountFromProto(a *pb.AccountInfo) *apiresource.Account {
	return &apiresource.Account{
		ID:        a.Id,
		Object:    constants.ObjectTypeAccount,
		Name:      a.Name,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
}
