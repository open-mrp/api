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

// Request to list volume discounts.
type ListVolumeDiscountsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of volume discounts, newest first.
//
// The search term matches the discount name, the name of a customer group it is scoped to, or the name of a product line it is scoped to. Customer portal users see only discounts with no customer-group restriction plus those scoped to a group their own account belongs to.
type ListVolumeDiscountsEndpoint struct{}

func (e *ListVolumeDiscountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListVolumeDiscountsRequest, *apiresource.List[apiresource.VolumeDiscount]] {
	return (&apiendpoint.APIEndpoint[*ListVolumeDiscountsRequest, *apiresource.List[apiresource.VolumeDiscount]]{
		Title:             "List Volume Discounts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/volume-discounts",
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListVolumeDiscountsRequest) (*apiresource.List[apiresource.VolumeDiscount], *apierror.APIError) {
			return svc.(VolumeDiscountSvc).ListVolumeDiscounts
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeVolumeDiscount,
			Fields:     []string{"customer_groups", "product_lines", "categories", "attributes", "acceptable_units"},
		}),
	})
}
