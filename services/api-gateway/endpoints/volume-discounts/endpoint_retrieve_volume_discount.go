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
)

// Request to retrieve a volume discount.
type RetrieveVolumeDiscountRequest struct {
	// Volume discount ID.
	VolumeDiscountID string `path:"id" validate:"required"`
}

// Returns a volume discount by ID.
//
// A customer or supplier portal user retrieves only a discount its own listing carries: one with no customer-group restriction, or one scoped to a group its account belongs to. Any other discount is reported as not found.
type RetrieveVolumeDiscountEndpoint struct{}

func (e *RetrieveVolumeDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveVolumeDiscountRequest, *apiresource.VolumeDiscount] {
	return (&apiendpoint.APIEndpoint[*RetrieveVolumeDiscountRequest, *apiresource.VolumeDiscount]{
		Title:             "Retrieve Volume Discount",
		Method:            http.MethodGet,
		Route:             "/v1/sales/volume-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ObjectType:        constants.ObjectTypeVolumeDiscount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveVolumeDiscountRequest) (*apiresource.VolumeDiscount, *apierror.APIError) {
			return svc.(VolumeDiscountSvc).GetVolumeDiscount
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeVolumeDiscount,
			Fields:     []string{"customer_groups", "product_lines", "categories", "categories.properties", "attributes", "acceptable_units"},
		}),
	})
}
