package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetInvoiceRequest is the request to retrieve a single invoice.
type GetInvoiceRequest struct {
	// The ID of the invoice to retrieve.
	InvoiceID string `path:"id" validate:"required"`
	// Sub-resources to include in the response.
	Includes []string `include:"true"`
}

type GetInvoiceEndpoint struct{}

func (e *GetInvoiceEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetInvoiceRequest, *apiresource.Invoice] {
	return &apiendpoint.APIEndpoint[*GetInvoiceRequest, *apiresource.Invoice]{
		Title:             "Get Invoice",
		Description:       "Returns a single invoice by its ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/invoices/{id}",
		Request:           &GetInvoiceRequest{},
		Response:          &apiresource.Invoice{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetInvoiceRequest) (*apiresource.Invoice, *apierror.APIError) {
			return svc.(InvoiceSvc).GetInvoice
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInvoice,
			Fields:     []string{"lines", "allocations"},
		}),
	}
}
