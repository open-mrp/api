package volumediscountep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
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

func (e *DeleteVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteVolumeDiscountRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteVolumeDiscountRequest, *object.Deleted]{
		Title:             "Delete Volume Discount",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/volume-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeVolumeDiscount,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteVolumeDiscountRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(VolumeDiscountSvc).DeleteVolumeDiscount, constants.ObjectTypeVolumeDiscount, "id")
		},
	})
}
