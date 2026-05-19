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

// Request to partially update a payment term.
type UpdatePaymentTermRequest struct {
	// Payment term ID.
	PaymentTermID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
}

var sampleUpdatePaymentTermRequest = &UpdatePaymentTermRequest{
	Name: new("Net 60"),
}

func (*UpdatePaymentTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePaymentTermRequest)
}

// Partially updates a payment term. Default payment terms cannot be updated.
type UpdatePaymentTermEndpoint struct{}

func (e *UpdatePaymentTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePaymentTermRequest, *apiresource.PaymentTerm] {
	return (&apiendpoint.APIEndpoint[*UpdatePaymentTermRequest, *apiresource.PaymentTerm]{
		Title:             "Update Payment Term",
		Method:            http.MethodPatch,
		Route:             "/v1/finance/payment-terms/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
			return svc.(PaymentTermSvc).UpdatePaymentTerm
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
