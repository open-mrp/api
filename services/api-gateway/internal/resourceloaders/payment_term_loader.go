package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var paymentTermLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.payment_term")

// LoadPaymentTerms fetches payment terms by ID via BatchGetPaymentTermsByIDs. Stashes owner_account_id in LoadMeta so the SubField closures can build the Owner shell and (on owner.account) write the loaded Account.
func LoadPaymentTerms(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, paymentTermLoaderTracer, "loader.payment_terms.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPaymentTermsByIDsResponse, error) {
			return coreClient.BatchGetPaymentTermsByIDs(ctx, &pb.BatchGetPaymentTermsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.PaymentTerms))
	for _, pt := range resp.PaymentTerms {
		out[pt.Id] = paymentTermFromProto(pt)
		var accountID string
		if pt.AccountId != nil {
			accountID = *pt.AccountId
		}
		meta.Set(constants.ObjectTypePaymentTerm, pt.Id, "owner_account_id", accountID)
	}
	return out, nil
}

func paymentTermFromProto(pt *pb.PaymentTermInfo) *apiresource.PaymentTerm {
	return &apiresource.PaymentTerm{
		ID:        pt.Id,
		Object:    constants.ObjectTypePaymentTerm,
		Name:      pt.Name,
		Status:    constants.PaymentTermStatus(pt.Status),
		CreatedAt: grpcutil.TimestampToTime(pt.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(pt.UpdatedAt),
	}
}
