package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a volume discount.
type RetrieveVolumeDiscountRequest struct {
	// Volume discount ID.
	VolumeDiscountID string `path:"id" validate:"required"`
}

// Returns a volume discount by ID.
type RetrieveVolumeDiscountEndpoint struct{}

func (e *RetrieveVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveVolumeDiscountRequest, *apiresource.VolumeDiscount] {
	return (&apiendpoint.APIEndpoint[*RetrieveVolumeDiscountRequest, *apiresource.VolumeDiscount]{
		Title:             "Retrieve Volume Discount",
		Method:            http.MethodGet,
		Route:             "/v1/sales/volume-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeVolumeDiscount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).GetVolumeDiscount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeVolumeDiscount,
			Fields:     []string{"customer_groups", "product_lines", "categories", "attributes", "acceptable_units"},
		}),
	})
}
