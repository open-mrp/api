package shippingcaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a shipping case by ID.
type GetShippingCaseRequest struct {
	// Shipping case ID.
	ShippingCaseID string `path:"id" validate:"required"`
}

type GetShippingCaseEndpoint struct{}

func (e *GetShippingCaseEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetShippingCaseRequest, *apiresource.ShippingCase] {
	return &apiendpoint.APIEndpoint[*GetShippingCaseRequest, *apiresource.ShippingCase]{
		Title:             "Get Shipping Case",
		Description:       "Returns a shipping case by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipping-cases/{id}",
		Request:           &GetShippingCaseRequest{},
		Response:          &apiresource.ShippingCase{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError) {
			return svc.(ShippingCaseSvc).GetShippingCase
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingCase,
			Fields:     []string{"carrier", "shipment", "freight_amount.unit", "freight_weight.unit"},
		}),
	}
}
