package ediep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to resubmit an invoice via EDI.
type ResubmitEDIInvoiceRequest struct {
	// Invoice ID.
	InvoiceID string `json:"invoice_id" validate:"required"`
}

var sampleResubmitEDIInvoiceRequest = &ResubmitEDIInvoiceRequest{
	InvoiceID: apiresource.SampleInvoiceID,
}

func (*ResubmitEDIInvoiceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleResubmitEDIInvoiceRequest)
}

// Resubmits an invoice via EDI. Fails if the invoice does not exist or EDI is not enabled on the account.
type ResubmitEDIInvoiceEndpoint struct{}

func (e *ResubmitEDIInvoiceEndpoint) Materialize() *apiendpoint.APIEndpoint[*ResubmitEDIInvoiceRequest, *apiresource.MessageResource] {
	return (&apiendpoint.APIEndpoint[*ResubmitEDIInvoiceRequest, *apiresource.MessageResource]{
		Title:             "Resubmit EDI Invoice",
		Method:            http.MethodPost,
		Route:             "/v1/operations/edi/actions/resubmit-invoice",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ResubmitEDIInvoiceRequest) (*apiresource.MessageResource, *apierror.APIError) {
			return svc.(EDISvc).ResubmitInvoice
		},
	})
}
