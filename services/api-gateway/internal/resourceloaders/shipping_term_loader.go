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
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.ShippingTerms))
	for _, st := range resp.ShippingTerms {
		out[st.Id] = shippingTermFromProto(st)

		var accountID string
		if st.AccountId != nil {
			accountID = *st.AccountId
		}
		meta.Set(constants.ObjectTypeShippingTerm, st.Id, "owner_account_id", accountID)

		if st.FlatRate != nil {
			meta.Set(constants.ObjectTypeShippingTerm, st.Id, "flat_rate_unit_id", st.FlatRate.UnitId)
		}
		if st.MinimumOrderValue != nil {
			meta.Set(constants.ObjectTypeShippingTerm, st.Id, "minimum_order_value_unit_id", st.MinimumOrderValue.UnitId)
		}

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
		FlatRate:          quantityFromProto(st.FlatRate),
		MinimumOrderValue: quantityFromProto(st.MinimumOrderValue),
		CreatedAt:         grpcutil.TimestampToTime(st.CreatedAt),
		UpdatedAt:         grpcutil.TimestampToTime(st.UpdatedAt),
	}
}

func quantityFromProto(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	norm := apiresource.NormalizeMonetaryQuantityValue(q.Value)
	return &apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        norm,
		DisplayValue: apiresource.FormatDisplayValue(norm, q.UnitAbbreviation, q.UnitType),
	}
}
