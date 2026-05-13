package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a volume discount.
type RetrieveVolumeDiscountRequest struct {
	// Volume discount ID.
	VolumeDiscountID string `path:"id" validate:"required"`
}

type RetrieveVolumeDiscountEndpoint struct{}

func (e *RetrieveVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveVolumeDiscountRequest, *apiresource.VolumeDiscount] {
	return &apiendpoint.APIEndpoint[*RetrieveVolumeDiscountRequest, *apiresource.VolumeDiscount]{
		Title:             "Retrieve Volume Discount",
		Description:       "Returns a volume discount by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/volume-discounts/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveVolumeDiscountRequest{},
		Response:          &apiresource.VolumeDiscount{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).GetVolumeDiscount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeVolumeDiscount,
			Fields:     []string{"customer_groups", "product_lines", "categories", "attributes", "acceptable_units"},
		}),
	}
}
