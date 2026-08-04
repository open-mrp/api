package shippingcaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a shipping case label URL.
type GetShippingCaseLabelRequest struct {
	// Shipping case ID.
	ShippingCaseID string `path:"id" validate:"required"`
}

// Returns a temporary download link for the shipping case's label image.
//
// The link expires one hour after it is issued, so fetch it when the label is about to be printed rather than storing it. No link is returned until a label has been generated for the case.
type GetShippingCaseLabelEndpoint struct{}

func (e *GetShippingCaseLabelEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetShippingCaseLabelRequest, *apiresource.ShippingCaseLabelURL] {
	return (&apiendpoint.APIEndpoint[*GetShippingCaseLabelRequest, *apiresource.ShippingCaseLabelURL]{
		Title:             "Get Shipping Case Label URL",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipping-cases/{id}/label",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		// GetShippingCaseLabel enforces shipments:read in the service (shipping cases
		// are a facet of shipments). Declared here to match that enforcement.
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainShipments, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetShippingCaseLabelRequest) (*apiresource.ShippingCaseLabelURL, *apierror.APIError) {
			return svc.(ShippingCaseSvc).GetShippingCaseLabel
		},
	})
}
