package supportrouteep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to designate the group conversation that handles a relationship's inbound support.
type SetSupportRouteRequest struct {
	// The customer account this route overrides for.
	//
	// Omit to set the account-level default applied to any customer.
	RelationAccountID field.Optional[string] `json:"relation_account_id,omitzero"`
	// The group conversation whose participants handle this relationship's support.
	//
	// It must be an existing group conversation in your account; a direct message, a system channel, or a conversation belonging to another account is rejected.
	GroupConversationID string `json:"group_conversation_id" validate:"required"`
}

var sampleSetSupportRouteRequest = &SetSupportRouteRequest{
	GroupConversationID: apiresource.SampleConversationID,
}

func (*SetSupportRouteRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetSupportRouteRequest)
}

// Designates the group conversation that handles a relationship's inbound support.
//
// The group's active people become the recipients seated on a customer's support thread when that customer opens one. A scope holds a single route, so setting one where a route already exists re-points it rather than adding a second.
//
// Configuring a route is what makes support reachable: until a customer's scope resolves to a route with at least one person in its group, that customer cannot open a new support thread. Re-pointing or clearing a route afterwards only affects threads opened from then on — people already seated on an open thread stay on it.
type SetSupportRouteEndpoint struct{}

func (e *SetSupportRouteEndpoint) Materialize() *apiendpoint.APIEndpoint[*SetSupportRouteRequest, *apiresource.SupportRoute] {
	return (&apiendpoint.APIEndpoint[*SetSupportRouteRequest, *apiresource.SupportRoute]{
		Title:               "Set Support Route",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/support-routes/actions/set",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeSupportRoute,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SetSupportRouteRequest) (*apiresource.SupportRoute, *apierror.APIError) {
			return svc.(SupportRouteSvc).SetSupportRoute
		},
	})
}
