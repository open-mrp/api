package shippingcaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a shipping case label URL.
type GetShippingCaseLabelRequest struct {
	// Shipping case ID.
	ShippingCaseID string `path:"id" validate:"required"`
}

// Returns a presigned URL for the shipping case's label image.
type GetShippingCaseLabelEndpoint struct{}

func (e *GetShippingCaseLabelEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetShippingCaseLabelRequest, *apiresource.ShippingCaseLabelURL] {
	return (&apiendpoint.APIEndpoint[*GetShippingCaseLabelRequest, *apiresource.ShippingCaseLabelURL]{
		Title:             "Get Shipping Case Label",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipping-cases/{id}/label",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetShippingCaseLabelRequest) (*apiresource.ShippingCaseLabelURL, *apierror.APIError) {
			return svc.(ShippingCaseSvc).GetShippingCaseLabel
		},
	})
}
