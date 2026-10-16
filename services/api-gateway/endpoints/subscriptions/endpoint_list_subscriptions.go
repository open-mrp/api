package subscriptionep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list your subscriptions.
type ListSubscriptionsRequest struct {
	object.ListRequest
	// Returns only your subscriptions to these records.
	RecordIDs []string `query:"record_ids" validate:"omitempty,max=100"`
	// Returns only your subscriptions to records of these types.
	RecordTypes []constants.RecordType `query:"record_types"`
	// Returns only subscriptions in these states. Omit it for both.
	Statuses []constants.RecordSubscriptionStatus `query:"statuses"`
}

// Returns a list of your subscriptions, newest first.
//
// Filter by `record_ids[]` to see whether you follow a record. Callers who aren't a person in the account, such as an API key, get an empty list.
type ListSubscriptionsEndpoint struct{}

func (e *ListSubscriptionsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSubscriptionsRequest, *apiresource.List[apiresource.Subscription]] {
	return (&apiendpoint.APIEndpoint[*ListSubscriptionsRequest, *apiresource.List[apiresource.Subscription]]{
		Title:             "List Subscriptions",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             subscriptionsRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSubscription,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSubscriptionsRequest) (*apiresource.List[apiresource.Subscription], *apierror.APIError) {
			return svc.(Svc).ListSubscriptions
		},
	})
}
