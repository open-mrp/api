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

// Request to create a payment term.
type CreatePaymentTermRequest struct {
	// Display name (e.g. `Net 30`).
	//
	// Must be unique among the payment terms visible to your account, including system defaults.
	Name string `json:"name" validate:"required,max=255"`
}

var sampleCreatePaymentTermRequest = &CreatePaymentTermRequest{
	Name: "Net 30",
}

func (*CreatePaymentTermRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePaymentTermRequest)
}

// Creates a payment term.
//
// The new term is owned by your account and starts with status `active`.
type CreatePaymentTermEndpoint struct{}

func (e *CreatePaymentTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePaymentTermRequest, *apiresource.PaymentTerm] {
	return (&apiendpoint.APIEndpoint[*CreatePaymentTermRequest, *apiresource.PaymentTerm]{
		Title:             "Create Payment Term",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/finance/payment-terms",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypePaymentTerm,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
			return svc.(PaymentTermSvc).CreatePaymentTerm
		},
		LocationFunc: func(resp *apiresource.PaymentTerm) string {
			return "/v1/finance/payment-terms/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
