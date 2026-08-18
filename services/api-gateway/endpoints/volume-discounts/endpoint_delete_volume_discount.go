package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a volume discount.
type DeleteVolumeDiscountRequest struct {
	// Volume discount ID.
	VolumeDiscountID string `path:"id" validate:"required"`
}

// Deletes a volume discount along with its tiers and scoping associations.
//
// Deletion is permanent; further requests against the deleted ID return an error.
//
// Order lines that have already been priced keep the unit price they were given; only lines priced after the deletion lose the discount.
type DeleteVolumeDiscountEndpoint struct{}

func (e *DeleteVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteVolumeDiscountRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteVolumeDiscountRequest, *apiresource.EmptyResource]{
		Title:             "Delete Volume Discount",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/volume-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteVolumeDiscountRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).DeleteVolumeDiscount
		},
	})
}
