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

// Request to remove a support route for a scope (account default or a specific relation).
type ClearSupportRouteRequest struct {
	// The customer account whose override to remove.
	//
	// Omit to clear the account-level default instead.
	RelationAccountID field.Optional[string] `json:"relation_account_id,omitzero"`
}

var sampleClearSupportRouteRequest = &ClearSupportRouteRequest{
	RelationAccountID: field.Some(apiresource.SampleCustomerID),
}

func (*ClearSupportRouteRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleClearSupportRouteRequest)
}

// Removes the support route configured for one scope in your account.
//
// Clearing a customer's override sends that customer back to the account-level default. Clearing the default leaves every customer without an override of their own unable to open a new support thread until a route is set again — threads that are already open keep working, and the people seated on them stay seated.
//
// Only the exact scope you name is cleared, and clearing a scope that has no route returns a not-found error.
type ClearSupportRouteEndpoint struct{}

func (e *ClearSupportRouteEndpoint) Materialize() *apiendpoint.APIEndpoint[*ClearSupportRouteRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*ClearSupportRouteRequest, *apiresource.EmptyResource]{
		Title:               "Clear Support Route",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/support-routes/actions/clear",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeSupportRoute,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ClearSupportRouteRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SupportRouteSvc).ClearSupportRoute
		},
	})
}
