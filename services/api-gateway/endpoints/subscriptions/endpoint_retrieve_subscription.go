package subscriptionep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve one of your subscriptions.
type RetrieveSubscriptionRequest struct {
	// Subscription ID.
	ID string `path:"id" validate:"required"`
}

// Retrieves one of your subscriptions.
type RetrieveSubscriptionEndpoint struct{}

func (e *RetrieveSubscriptionEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveSubscriptionRequest, *apiresource.Subscription] {
	return (&apiendpoint.APIEndpoint[*RetrieveSubscriptionRequest, *apiresource.Subscription]{
		Title:             "Retrieve Subscription",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             subscriptionsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSubscription,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSubscriptionRequest) (*apiresource.Subscription, *apierror.APIError) {
			return svc.(Svc).GetSubscription
		},
	})
}
