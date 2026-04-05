package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetVolumeDiscountRequest is the request to retrieve a single volume discount.
type GetVolumeDiscountRequest struct {
	// The ID of the volume discount to retrieve.
	VolumeDiscountID string `path:"id" validate:"required"`
}

type GetVolumeDiscountEndpoint struct{}

func (e *GetVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetVolumeDiscountRequest, *apiresource.VolumeDiscount] {
	return &apiendpoint.APIEndpoint[*GetVolumeDiscountRequest, *apiresource.VolumeDiscount]{
		Title:             "Get Volume Discount",
		Description:       "Returns a single volume discount by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/volume-discounts/{id}",
		ContentType:       "application/json",
		Request:           &GetVolumeDiscountRequest{},
		Response:          &apiresource.VolumeDiscount{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).GetVolumeDiscount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeVolumeDiscount,
			Fields:     []string{"customer_groups", "product_lines", "categories", "attributes", "acceptable_units"},
		}),
	}
}
