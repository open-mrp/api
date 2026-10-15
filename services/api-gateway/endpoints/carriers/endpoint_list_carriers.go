package carrierep

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

// Request to list carriers.
type ListCarriersRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return carriers with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
}

// Returns a list of carriers.
//
// Includes the carriers your account created and the system carriers every account shares. Carriers are sorted by creation date, newest first.
type ListCarriersEndpoint struct{}

func (e *ListCarriersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListCarriersRequest, *apiresource.List[apiresource.Carrier]] {
	return (&apiendpoint.APIEndpoint[*ListCarriersRequest, *apiresource.List[apiresource.Carrier]]{
		Title:             "List Carriers",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ObjectType:        constants.ObjectTypeCarrier,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListCarriersRequest) (*apiresource.List[apiresource.Carrier], *apierror.APIError) {
			return svc.(CarrierSvc).ListCarriers
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"default_service_level", "owner", "owner.account", "service_levels"},
		}),
	})
}
