package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// A notification recipient to configure for a customer.
type NotificationRecipientInput struct {
	// ID of the account user to receive the notifications.
	//
	// Must be an account user on the customer's own account.
	AccountUserID string `json:"account_user_id" validate:"required"`
	// Order notification types this recipient should receive.
	//
	// Only `order_acknowledgement` and `invoice` can be set here; any other type is rejected.
	NotificationTypes []constants.AccountRelationNotificationType `json:"notification_types" validate:"required,min=1"`
}

// Request to replace a customer's default order-notification recipients.
type UpdateNotificationRecipientsRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
	// The complete desired set of default notification recipients.
	//
	// Any recipient not included is removed; send an empty list to remove them all.
	Recipients []NotificationRecipientInput `json:"recipients" validate:"dive"`
}

var sampleUpdateNotificationRecipientsRequest = &UpdateNotificationRecipientsRequest{
	Recipients: []NotificationRecipientInput{
		{
			AccountUserID: apiresource.SampleAccountUserID,
			NotificationTypes: []constants.AccountRelationNotificationType{
				constants.AccountRelationNotificationTypeOrderAcknowledgement,
				constants.AccountRelationNotificationTypeInvoice,
			},
		},
	},
}

func (*UpdateNotificationRecipientsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateNotificationRecipientsRequest)
}

// Replaces the account users configured to receive order acknowledgement and invoice emails on this customer's orders.
//
// The provided list is the complete set of recipients; any recipient not included is removed. Only the order acknowledgement and invoice notification types can be managed here — purchase-order submission preferences on the same relationship are left untouched, and still appear in the returned recipients.
type UpdateNotificationRecipientsEndpoint struct{}

func (e *UpdateNotificationRecipientsEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateNotificationRecipientsRequest, *apiresource.List[apiresource.OrderNotificationRecipient]] {
	return (&apiendpoint.APIEndpoint[*UpdateNotificationRecipientsRequest, *apiresource.List[apiresource.OrderNotificationRecipient]]{
		Title:               "Update Customer Notification Recipients",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/sales/customers/{id}/notification-recipients",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateNotificationRecipientsRequest) (*apiresource.List[apiresource.OrderNotificationRecipient], *apierror.APIError) {
			return svc.(CustomerSvc).UpdateNotificationRecipients
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeOrderNotificationRecipient,
			Fields:     []string{"account_user"},
		}),
	})
}
