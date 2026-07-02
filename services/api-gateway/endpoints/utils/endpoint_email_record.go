package utilsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to email a record to its configured recipients.
type EmailRecordRequest struct {
	// ID of the record to email.
	ID string `json:"id" validate:"required"`
	// The type of record to email.
	//
	// - `invoice`: emails the invoice to the invoice's email recipients.
	// - `sales_order`: sends an order acknowledgement to the order's acknowledgement recipients.
	// - `purchase_order`: sends the purchase order submission to the order's submission recipients.
	Type string `json:"type" validate:"required"`
}

var sampleEmailRecordRequest = &EmailRecordRequest{
	ID:   apiresource.SampleInvoiceID,
	Type: "invoice",
}

func (*EmailRecordRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleEmailRecordRequest)
}

// Emails a record (invoice, sales order, or purchase order) to its configured recipients and marks the record as sent.
//
// Delivery is asynchronous: the endpoint returns `202 Accepted` once the email is queued. If the record has no configured recipients, the request succeeds without sending an email.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInvoices, Action: types.ActionRead},
			{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead},
			{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *EmailRecordRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(UtilsSvc).EmailRecord
		},
	})
}
