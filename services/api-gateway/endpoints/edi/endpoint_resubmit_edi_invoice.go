package ediep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ResubmitEDIInvoiceRequest is the request to resubmit an invoice via EDI.
type ResubmitEDIInvoiceRequest struct {
	// The ID of the invoice to resubmit.
	InvoiceID string `json:"invoice_id" validate:"required"`
}

var exampleResubmitEDIInvoiceRequest = &ResubmitEDIInvoiceRequest{
	InvoiceID: "inv_abc123",
}

func (*ResubmitEDIInvoiceRequest) SchemaExample() any {
	return exampleResubmitEDIInvoiceRequest
}

type ResubmitEDIInvoiceEndpoint struct{}

func (e *ResubmitEDIInvoiceEndpoint) Materialize() *apiendpoint.APIEndpoint[*ResubmitEDIInvoiceRequest, *apiresource.MessageResource] {
	return &apiendpoint.APIEndpoint[*ResubmitEDIInvoiceRequest, *apiresource.MessageResource]{
		Title:             "Resubmit EDI Invoice",
		Description:       "Resubmits an invoice via EDI. Requires the invoice to exist and EDI to be enabled on the account.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/edi/actions/resubmit-invoice",
		ContentType:       "application/json",
		Request:           &ResubmitEDIInvoiceRequest{},
		Response:          &apiresource.MessageResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ResubmitEDIInvoiceRequest) (*apiresource.MessageResource, *apierror.APIError) {
			return svc.(EDISvc).ResubmitInvoice
		},
	}
}
