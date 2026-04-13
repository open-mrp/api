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
type GetPaymentTermRequest struct {
	// Payment term ID.
	PaymentTermID string `path:"id" validate:"required"`
}

type GetPaymentTermEndpoint struct{}

func (e *GetPaymentTermEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPaymentTermRequest, *apiresource.PaymentTerm] {
	return &apiendpoint.APIEndpoint[*GetPaymentTermRequest, *apiresource.PaymentTerm]{
		Title:             "Get Payment Term",
		Description:       "Returns a payment term by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/finance/payment-terms/{id}",
		ContentType:       "application/json",
		Request:           &GetPaymentTermRequest{},
		Response:          &apiresource.PaymentTerm{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
			return svc.(PaymentTermSvc).GetPaymentTerm
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePaymentTerm,
			Fields:     []string{"owner", "owner.account"},
		}),
	}
}
