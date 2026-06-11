package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update an invoice.
type UpdateInvoiceRequest struct {
	// Invoice ID.
	InvoiceID string `path:"id" validate:"required"`
	// Note to attach to the invoice.
	Note field.Optional[string] `json:"note,omitzero"`
	// Whether the invoice has been sent to the customer.
	HasBeenSent field.Optional[bool] `json:"has_been_sent,omitzero"`
	// Whether the invoice has been sent via EDI.
	IsEdiSent field.Optional[bool] `json:"is_edi_sent,omitzero"`
	// Whether the invoice has been paid in full.
	//
	// Setting this to `true` marks the invoice as paid regardless of recorded allocations, which updates the invoice's `payment_status` and removes it from receivables listings.
	IsPaidInFull field.Optional[bool] `json:"is_paid_in_full,omitzero"`
}

var sampleUpdateInvoiceNote = "Payment received via wire transfer"
var sampleUpdateInvoiceHasBeenSent = true
var sampleUpdateInvoiceRequest = &UpdateInvoiceRequest{
	Note:        field.Some(sampleUpdateInvoiceNote),
	HasBeenSent: field.Some(sampleUpdateInvoiceHasBeenSent),
}

func (*UpdateInvoiceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateInvoiceRequest)
}

// Partially updates an invoice.
type UpdateInvoiceEndpoint struct{}

func (e *UpdateInvoiceEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateInvoiceRequest, *apiresource.Invoice] {
	return (&apiendpoint.APIEndpoint[*UpdateInvoiceRequest, *apiresource.Invoice]{
		Title:             "Update Invoice",
		Method:            http.MethodPatch,
		Route:             "/v1/finance/invoices/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeInvoice,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateInvoiceRequest) (*apiresource.Invoice, *apierror.APIError) {
			return svc.(InvoiceSvc).UpdateInvoice
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInvoice,
			Fields:     []string{"customer", "order", "shipment", "billing_address", "payment_term", "lines", "allocations"},
		}),
	})
}
