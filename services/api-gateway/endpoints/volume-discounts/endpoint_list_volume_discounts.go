package volumediscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list volume discounts.
type ListVolumeDiscountsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of volume discounts for the target account.
type ListVolumeDiscountsEndpoint struct{}

func (e *ListVolumeDiscountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListVolumeDiscountsRequest, *apiresource.List[apiresource.VolumeDiscount]] {
	return (&apiendpoint.APIEndpoint[*ListVolumeDiscountsRequest, *apiresource.List[apiresource.VolumeDiscount]]{
		Title:             "List Volume Discounts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/volume-discounts",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
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
	})
}
