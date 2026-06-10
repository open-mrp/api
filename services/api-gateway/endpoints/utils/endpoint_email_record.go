package utilsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to email a record to its configured recipients.
type EmailRecordRequest struct {
	// Record ID.
	ID string `json:"id" validate:"required"`
	// Record type: invoice, sales_order, or purchase_order.
	Type string `json:"type" validate:"required"`
}

var sampleEmailRecordRequest = &EmailRecordRequest{
	ID:   apiresource.SampleInvoiceID,
	Type: "invoice",
}

func (*EmailRecordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleEmailRecordRequest)
}

// Emails a record (invoice, sales order, or purchase order) to the configured recipients as a PDF attachment.
type EmailRecordEndpoint struct{}

func (e *EmailRecordEndpoint) Materialize() *apiendpoint.APIEndpoint[*EmailRecordRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*EmailRecordRequest, *apiresource.EmptyResource]{
		Title:             "Email Record",
		Method:            http.MethodPost,
		Route:             "/v1/core/actions/email-record",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *EmailRecordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UtilsSvc).EmailRecord
		},
	})
}
