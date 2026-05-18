package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update an invoice.
type UpdateInvoiceRequest struct {
	// Invoice ID.
	InvoiceID string `path:"id" validate:"required"`
	// Note to attach to the invoice.
	Note *string `json:"note,omitempty" nullable:"false"`
	// Whether the invoice has been sent.
	HasBeenSent *bool `json:"has_been_sent,omitempty" nullable:"false"`
	// Whether the invoice has been sent via EDI.
	IsEdiSent *bool `json:"is_edi_sent,omitempty" nullable:"false"`
	// Whether the invoice has been paid in full.
	IsPaidInFull *bool `json:"is_paid_in_full,omitempty" nullable:"false"`
}

var sampleUpdateInvoiceNote = "Payment received via wire transfer"
var sampleUpdateInvoiceHasBeenSent = true
var sampleUpdateInvoiceRequest = &UpdateInvoiceRequest{
	Note:        &sampleUpdateInvoiceNote,
	HasBeenSent: &sampleUpdateInvoiceHasBeenSent,
}

func (*UpdateInvoiceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateInvoiceRequest)
}

// Partially updates an invoice.
type UpdateInvoiceEndpoint struct{}

func (e *UpdateInvoiceEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateInvoiceRequest, *apiresource.InvoiceSummary] {
	return (&apiendpoint.APIEndpoint[*UpdateInvoiceRequest, *apiresource.InvoiceSummary]{
		Title:             "Update Invoice",
		Method:            http.MethodPatch,
		Route:             "/v1/finance/invoices/{id}",
		ContentType:       "application/json",
		Request:           &UpdateInvoiceRequest{},
		Response:          &apiresource.InvoiceSummary{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateInvoiceRequest) (*apiresource.InvoiceSummary, *apierror.APIError) {
			return svc.(InvoiceSvc).UpdateInvoice
		},
	}).WithDocSource(e)
}
