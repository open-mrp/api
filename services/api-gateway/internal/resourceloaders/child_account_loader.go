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

var childAccountLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.child_account")

// LoadChildAccounts fetches child account relations by relation_id via BatchGetChildAccountsByIDs. The inline Account is built directly from the proto's denormalized account fields — no expandable sub-resources.
func LoadChildAccounts(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, childAccountLoaderTracer, "loader.child_accounts.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetChildAccountsByIDsResponse, error) {
			return coreClient.BatchGetChildAccountsByIDs(ctx, &pb.BatchGetChildAccountsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Items))
	for _, ca := range resp.Items {
		out[ca.RelationId] = ChildAccountFromProto(ca)
	}
	return out, nil
}

// ChildAccountFromProto maps the gRPC ChildAccountProto to the apiresource shape. Exported so endpoint service methods that already hold a proto response (Add returns the resource directly) can reuse it.
func ChildAccountFromProto(ca *pb.ChildAccountProto) *apiresource.ChildAccount {
	out := &apiresource.ChildAccount{
		ID:     ca.RelationId,
		Object: constants.ObjectTypeChildAccount,
		Account: &apiresource.Account{
			ID:     ca.AccountId,
			Object: constants.ObjectTypeAccount,
			Name:   ca.AccountName,
		},
		Email:     ca.Email,
		CreatedAt: grpcutil.TimestampToTime(ca.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(ca.UpdatedAt),
	}
	if ca.ExternalNumber != "" {
		s := ca.ExternalNumber
		out.ExternalNumber = &s
	}
	return out
}
