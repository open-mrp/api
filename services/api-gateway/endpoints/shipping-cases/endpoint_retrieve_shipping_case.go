package shippingcaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve a shipping case by ID.
type RetrieveShippingCaseRequest struct {
	// Shipping case ID.
	ShippingCaseID string `path:"id" validate:"required"`
}

// Returns a shipping case by ID.
type RetrieveShippingCaseEndpoint struct{}

func (e *RetrieveShippingCaseEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShippingCaseRequest, *apiresource.ShippingCase] {
	return (&apiendpoint.APIEndpoint[*RetrieveShippingCaseRequest, *apiresource.ShippingCase]{
		Title:             "Retrieve Shipping Case",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipping-cases/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		// GetShippingCase enforces shipments:read in the service (shipping cases are
		// a facet of shipments). Declared here to match that enforcement.
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeShippingCase,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError) {
			return svc.(ShippingCaseSvc).GetShippingCase
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingCase,
			Fields:     []string{"carrier", "shipment", "freight_amount", "freight_amount.unit", "freight_weight", "freight_weight.unit"},
		}),
	})
}
