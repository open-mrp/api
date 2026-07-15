package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list a customer's default order-notification recipients.
type ListNotificationRecipientsRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
}

// Returns the account users configured to receive order acknowledgement and invoice emails on this customer's orders, with the notification types each receives.
//
// These are the defaults used to pre-populate notification recipients when a new order is placed for the customer.
type ListNotificationRecipientsEndpoint struct{}

func (e *ListNotificationRecipientsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListNotificationRecipientsRequest, *apiresource.List[apiresource.OrderNotificationRecipient]] {
	return (&apiendpoint.APIEndpoint[*ListNotificationRecipientsRequest, *apiresource.List[apiresource.OrderNotificationRecipient]]{
		Title:               "List Customer Notification Recipients",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/sales/customers/{id}/notification-recipients",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListNotificationRecipientsRequest) (*apiresource.List[apiresource.OrderNotificationRecipient], *apierror.APIError) {
			return svc.(CustomerSvc).ListNotificationRecipients
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeOrderNotificationRecipient,
			Fields:     []string{"account_user"},
		}),
	})
}
