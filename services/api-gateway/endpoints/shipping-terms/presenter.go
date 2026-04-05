package shippingtermep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func quantityPresenter(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	norm := apiresource.NormalizeMonetaryQuantityValue(q.Value)
	return &apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        norm,
		DisplayValue: apiresource.FormatDisplayValue(norm, q.UnitAbbreviation, q.UnitType),
		Unit:         nil, // Populated via include expansion
	}
}

func ShippingTermPresenter(st *pb.ShippingTermInfo) apiresource.ShippingTerm {
	if st == nil {
		return apiresource.ShippingTerm{}
	}

	freeShippingServiceLevels := make([]apiresource.ServiceLevel, len(st.FreeShippingServiceLevelIds))
	for i, id := range st.FreeShippingServiceLevelIds {
		freeShippingServiceLevels[i] = apiresource.ServiceLevel{
			ID:     id,
			Object: constants.ObjectTypeServiceLevel,
		}
	}

	return apiresource.ShippingTerm{
		ID:                        st.Id,
		Object:                    constants.ObjectTypeShippingTerm,
		Name:                      st.Name,
		Type:                      constants.ShippingTermType(st.Type),
		FlatRate:                  quantityPresenter(st.FlatRate),
		MinimumOrderValue:         quantityPresenter(st.MinimumOrderValue),
		FreeShippingServiceLevels: apiresource.NewList(freeShippingServiceLevels, apiresource.PageInfo{}),
		Owner:                     apiresource.NewOwner(st.AccountId),
		CreatedAt:                 grpcutil.TimestampToTime(st.CreatedAt),
		UpdatedAt:                 grpcutil.TimestampToTime(st.UpdatedAt),
	}
}

func ShippingTermListPresenter(resp *pb.ListShippingTermsResponse) *apiresource.List[apiresource.ShippingTerm] {
	if resp == nil {
		return apiresource.NewList[apiresource.ShippingTerm](nil, apiresource.PageInfo{})
	}

	shippingTerms := make([]apiresource.ShippingTerm, len(resp.ShippingTerms))
	for i, st := range resp.ShippingTerms {
		shippingTerms[i] = ShippingTermPresenter(st)
	}

	return apiresource.NewList(shippingTerms, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
