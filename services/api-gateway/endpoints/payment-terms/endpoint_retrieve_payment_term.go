package paymenttermep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a payment term.
type RetrievePaymentTermRequest struct {
	// Payment term ID.
	PaymentTermID string `path:"id" validate:"required"`
}

// Returns a payment term by ID.
type RetrievePaymentTermEndpoint struct{}

func (e *RetrievePaymentTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePaymentTermRequest, *apiresource.PaymentTerm] {
	return (&apiendpoint.APIEndpoint[*RetrievePaymentTermRequest, *apiresource.PaymentTerm]{
		Title:             "Retrieve Payment Term",
		Method:            http.MethodGet,
		Route:             "/v1/finance/payment-terms/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
			return svc.(PaymentTermSvc).GetPaymentTerm
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
