package receivableep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to email a statement of account for a specific customer.
type EmailReceivablesForCustomerRequest struct {
	// Customer account ID.
	AccountID string `json:"-" path:"account_id" validate:"required"`
	// Email addresses to send the statement of account to.
	RecipientEmails []string `json:"recipient_emails" validate:"required,min=1"`
}

var sampleEmailReceivablesForCustomerRequest = &EmailReceivablesForCustomerRequest{
	RecipientEmails: []string{apiresource.SampleUserEmail},
}

func (*EmailReceivablesForCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleEmailReceivablesForCustomerRequest)
}

// Emails a statement of account for a specific customer to the provided recipients.
//
// The email carries an Excel attachment listing the customer's outstanding receivables and open credits.
type EmailReceivablesForCustomerEndpoint struct{}

func (e *EmailReceivablesForCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*EmailReceivablesForCustomerRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*EmailReceivablesForCustomerRequest, *apiresource.EmptyResource]{
		Title:             "Email Receivables for Customer",
		Method:            http.MethodPost,
		Route:             "/v1/finance/accounts/{account_id}/actions/email-receivables",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *EmailReceivablesForCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ReceivableSvc).EmailReceivablesForCustomer
		},
	})
}
