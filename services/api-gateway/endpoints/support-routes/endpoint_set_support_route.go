package supportrouteep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to designate the group conversation that handles a relationship's inbound support.
type SetSupportRouteRequest struct {
	// The customer account this route overrides for.
	//
	// Omit to set the account-level default applied to any customer.
	RelationAccountID field.Optional[string] `json:"relation_account_id,omitzero"`
	// The group conversation whose participants receive this relationship's support.
	GroupConversationID string `json:"group_conversation_id" validate:"required"`
}

var sampleSetSupportRouteRequest = &SetSupportRouteRequest{
	GroupConversationID: apiresource.SampleConversationID,
}

func (*SetSupportRouteRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetSupportRouteRequest)
}

// Designates (or re-points) the group conversation that handles a relationship's inbound support.
//
// Its participants become the deterministic recipients seated on the customer's support thread. The target must be an existing group conversation in the caller's account.
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
