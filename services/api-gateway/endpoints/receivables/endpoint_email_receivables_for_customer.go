package receivableep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// EmailReceivablesForCustomerRequest is the request to email receivable entries for a specific customer.
type EmailReceivablesForCustomerRequest struct {
	// The customer account ID.
	AccountID string `json:"-" path:"account_id" validate:"required"`
	// The email addresses to send the receivables report to.
	RecipientEmails []string `json:"recipient_emails" validate:"required,min=1"`
}

var sampleEmailReceivablesForCustomerRequest = &EmailReceivablesForCustomerRequest{
	RecipientEmails: []string{apiresource.SampleUserEmail},
}

func (*EmailReceivablesForCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleEmailReceivablesForCustomerRequest)
}

type EmailReceivablesForCustomerEndpoint struct{}

func (e *EmailReceivablesForCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*EmailReceivablesForCustomerRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*EmailReceivablesForCustomerRequest, *apiresource.EmptyResource]{
		Title:             "Email Receivables for Customer",
		Description:       "Sends a receivables report for a specific customer account to the provided email addresses.",
		Method:            http.MethodPost,
		Route:             "/v1/finance/accounts/{account_id}/actions/email-receivables",
		ContentType:       "application/json",
		Request:           &EmailReceivablesForCustomerRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *EmailReceivablesForCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ReceivableSvc).EmailReceivablesForCustomer
		},
	}
}
