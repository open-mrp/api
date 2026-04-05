package paymenttermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdatePaymentTermRequest is the request to partially update a payment term.
type UpdatePaymentTermRequest struct {
	// The ID of the payment term to update.
	PaymentTermID string `path:"id" validate:"required"`
	// The display name of the payment term.
	Name *string `json:"name,omitempty"`
}

var sampleUpdatePaymentTermRequest = &UpdatePaymentTermRequest{
	Name: new("Net 60"),
}

func (*UpdatePaymentTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePaymentTermRequest)
}

type UpdatePaymentTermEndpoint struct{}

func (e *UpdatePaymentTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePaymentTermRequest, *apiresource.PaymentTerm] {
	return &apiendpoint.APIEndpoint[*UpdatePaymentTermRequest, *apiresource.PaymentTerm]{
		Title:             "Update Payment Term",
		Description:       "Partially updates a payment term. Default payment terms cannot be updated.",
		Method:            http.MethodPatch,
		Route:             "/v1/finance/payment-terms/{id}",
		ContentType:       "application/json",
		Request:           &UpdatePaymentTermRequest{},
		Response:          &apiresource.PaymentTerm{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
			return svc.(PaymentTermSvc).UpdatePaymentTerm
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner"},
		}),
	}
}
