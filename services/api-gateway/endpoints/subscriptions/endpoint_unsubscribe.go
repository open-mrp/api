package subscriptionep

import (
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	"github.com/open-mrp/api/services/api-gateway/pkg/lifecycle"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Unsubscribes you from a record, so you're no longer notified about it.
//
// You stay unsubscribed until you subscribe again: creating, being assigned to or being mentioned about the record doesn't subscribe you automatically. Unsubscribing again changes nothing.
type UnsubscribeEndpoint struct{}

func (e *UnsubscribeEndpoint) Materialize() *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.Subscription] {
	return lifecycle.Action("Unsubscribe", subscriptionsRoute+"/{id}/actions/unsubscribe", constants.ObjectTypeSubscription, authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		func(svc any) apiendpoint.ServiceHandler[*lifecycle.Request, *apiresource.Subscription] {
			return svc.(Svc).Unsubscribe
		})
}
