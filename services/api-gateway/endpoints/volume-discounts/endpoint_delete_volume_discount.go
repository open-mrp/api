package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a volume discount.
type DeleteVolumeDiscountRequest struct {
	// Volume discount ID.
	VolumeDiscountID string `path:"id" validate:"required"`
}

// Deletes a volume discount and all associated tiers and relations.
type DeleteVolumeDiscountEndpoint struct{}

func (e *DeleteVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteVolumeDiscountRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteVolumeDiscountRequest, *apiresource.EmptyResource]{
		Title:             "Delete Volume Discount",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/volume-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteVolumeDiscountRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).DeleteVolumeDiscount
		},
	})
}
