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
type RetrieveShippingCaseRequest struct {
	// Shipping case ID.
	ShippingCaseID string `path:"id" validate:"required"`
}

type RetrieveShippingCaseEndpoint struct{}

func (e *RetrieveShippingCaseEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveShippingCaseRequest, *apiresource.ShippingCase] {
	return &apiendpoint.APIEndpoint[*RetrieveShippingCaseRequest, *apiresource.ShippingCase]{
		Title:             "Retrieve Shipping Case",
		Description:       "Returns a shipping case by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/shipping-cases/{id}",
		Request:           &RetrieveShippingCaseRequest{},
		Response:          &apiresource.ShippingCase{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError) {
			return svc.(ShippingCaseSvc).GetShippingCase
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeShippingCase,
			Fields:     []string{"carrier", "shipment", "freight_amount.unit", "freight_weight.unit"},
		}),
	}
}
