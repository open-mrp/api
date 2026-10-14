package resourceloaders

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

var shippingTermLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.shipping_term")

func LoadShippingTerms(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, shippingTermLoaderTracer, "loader.shipping_terms.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetShippingTermsByIDsResponse, error) {
			return coreClient.BatchGetShippingTermsByIDs(ctx, &pb.BatchGetShippingTermsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.ShippingTerms))
	for _, st := range resp.ShippingTerms {
		out[st.Id] = shippingTermFromProto(st)

		var accountID string
		if st.AccountId != nil {
			accountID = *st.AccountId
		}
		meta.Set(constants.ObjectTypeShippingTerm, st.Id, "owner_account_id", accountID)

		slIDs := make([]string, len(st.FreeShippingServiceLevels))
		for i, sl := range st.FreeShippingServiceLevels {
			slIDs[i] = sl.Id
		}
		meta.Set(constants.ObjectTypeShippingTerm, st.Id, "free_shipping_service_level_ids", slIDs)
	}
	return out, nil
}

func shippingTermFromProto(st *pb.ShippingTermInfo) *apiresource.ShippingTerm {
	return &apiresource.ShippingTerm{
		ID:                st.Id,
		Object:            constants.ObjectTypeShippingTerm,
		Name:              st.Name,
		Type:              constants.ShippingTermType(st.Type),
		FlatRate:          MoneyFromProto(st.FlatRate),
		MinimumOrderValue: MoneyFromProto(st.MinimumOrderValue),
		CreatedAt:         grpcutil.TimestampToTime(st.CreatedAt),
		UpdatedAt:         grpcutil.TimestampToTime(st.UpdatedAt),
	}
}

