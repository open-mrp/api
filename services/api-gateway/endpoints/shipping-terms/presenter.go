package shippingtermep

import (
	"context"

	servicelevelep "github.com/augno/api/services/api-gateway/endpoints/service-levels"
	unitep "github.com/augno/api/services/api-gateway/endpoints/units"
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
	var unit *apiresource.Unit
	if ud := q.GetUnitDetail(); ud != nil {
		u := unitep.UnitPresenter(ud, nil)
		unit = &u
	} else {
		unit = &apiresource.Unit{
			ID:           q.UnitId,
			Object:       constants.ObjectTypeUnit,
			Name:         q.UnitName,
			Abbreviation: q.UnitAbbreviation,
			Type:         constants.UnitType(q.UnitType),
		}
	}
	return &apiresource.Quantity{
		ID:           q.Id,
		Object:       constants.ObjectTypeQuantity,
		Value:        norm,
		DisplayValue: apiresource.FormatDisplayValue(norm, q.UnitAbbreviation, q.UnitType),
		Unit:         unit,
	}
}

func ShippingTermPresenter(st *pb.ShippingTermInfo, ownerAccount *apiresource.Account) apiresource.ShippingTerm {
	if st == nil {
		return apiresource.ShippingTerm{}
	}

	levels := st.GetFreeShippingServiceLevels()
	freeShippingServiceLevels := make([]apiresource.ServiceLevel, len(levels))
	for i, sl := range levels {
		freeShippingServiceLevels[i] = servicelevelep.ServiceLevelPresenter(sl, nil)
	}

	return apiresource.ShippingTerm{
		ID:                        st.Id,
		Object:                    constants.ObjectTypeShippingTerm,
		Name:                      st.Name,
		Type:                      constants.ShippingTermType(st.Type),
		FlatRate:                  quantityPresenter(st.FlatRate),
		MinimumOrderValue:         quantityPresenter(st.MinimumOrderValue),
		FreeShippingServiceLevels: apiresource.NewList(freeShippingServiceLevels, apiresource.PageInfo{}),
		Owner:                     apiresource.NewOwnerWithAccount(st.AccountId, ownerAccount),
		CreatedAt:                 grpcutil.TimestampToTime(st.CreatedAt),
		UpdatedAt:                 grpcutil.TimestampToTime(st.UpdatedAt),
	}
}

func ShippingTermListPresenter(ctx context.Context, resp *pb.ListShippingTermsResponse, ownerAccount *apiresource.Account) *apiresource.List[apiresource.ShippingTerm] {
	if resp == nil {
		return apiresource.NewList[apiresource.ShippingTerm](nil, apiresource.PageInfo{})
	}

	shippingTerms := make([]apiresource.ShippingTerm, len(resp.ShippingTerms))
	for i, st := range resp.ShippingTerms {
		shippingTerms[i] = ShippingTermPresenter(st, ownerAccount)
	}

	return apiresource.NewList(shippingTerms, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
