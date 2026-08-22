package ediep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to resubmit an invoice via EDI.
type ResubmitEDIInvoiceRequest struct {
	// ID of the invoice to resubmit.
	InvoiceID string `json:"invoice_id" validate:"required"`
}

var sampleResubmitEDIInvoiceRequest = &ResubmitEDIInvoiceRequest{
	InvoiceID: apiresource.SampleInvoiceID,
}

func (*ResubmitEDIInvoiceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleResubmitEDIInvoiceRequest)
}

// Resubmits an invoice over EDI.
//
// Use this to send an invoice again when its earlier EDI submission did not reach the customer.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInvoices, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ResubmitEDIInvoiceRequest) (*apiresource.MessageResource, *apierror.APIError) {
			return svc.(EDISvc).ResubmitInvoice
		},
	})
}
