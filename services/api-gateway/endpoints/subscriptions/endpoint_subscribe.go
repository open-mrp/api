package subscriptionep

import (
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	"github.com/open-mrp/api/services/api-gateway/pkg/lifecycle"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Subscribes you to a record again after you unsubscribed.
//
// The subscription's reason becomes `manual`. Subscribing again while subscribed changes nothing.
type SubscribeEndpoint struct{}

func (e *SubscribeEndpoint) Materialize() *apiendpoint.APIEndpoint[*lifecycle.Request, *apiresource.Subscription] {
	return lifecycle.Action("Subscribe", subscriptionsRoute+"/{id}/actions/subscribe", constants.ObjectTypeSubscription, authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}}},
		func(svc any) apiendpoint.ServiceHandler[*lifecycle.Request, *apiresource.Subscription] {
			return svc.(Svc).Subscribe
		})
}
