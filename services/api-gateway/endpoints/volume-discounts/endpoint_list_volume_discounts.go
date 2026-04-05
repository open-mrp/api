package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListVolumeDiscountsRequest is the request to list volume discounts.
type ListVolumeDiscountsRequest struct {
	apiresource.PaginationRequest
}

type ListVolumeDiscountsEndpoint struct{}

func (e *ListVolumeDiscountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListVolumeDiscountsRequest, *apiresource.List[apiresource.VolumeDiscount]] {
	return &apiendpoint.APIEndpoint[*ListVolumeDiscountsRequest, *apiresource.List[apiresource.VolumeDiscount]]{
		Title:             "List Volume Discounts",
		Description:       "Returns a paginated list of volume discounts for the target account.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/volume-discounts",
		Request:           &ListVolumeDiscountsRequest{},
		Response:          &apiresource.List[apiresource.VolumeDiscount]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListVolumeDiscountsRequest) (*apiresource.List[apiresource.VolumeDiscount], *apierror.APIError) {
			return svc.(VolumeDiscountSvc).ListVolumeDiscounts
		},
	}
}
