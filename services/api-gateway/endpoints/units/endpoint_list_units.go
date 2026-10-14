package unitep

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

// Request to list units.
type ListUnitsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Filter by unit dimension.
	Type *constants.UnitType `query:"type"`
	// Return only units that belong to at least one of the given unit groups.
	UnitGroupIDs []string `query:"unit_group_ids"`
}

// Returns a paginated list of units for the current account, including both account-owned and global system units.
type ListUnitsEndpoint struct{}

func (e *ListUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListUnitsRequest, *apiresource.List[apiresource.Unit]] {
	return (&apiendpoint.APIEndpoint[*ListUnitsRequest, *apiresource.List[apiresource.Unit]]{
		Title:             "List Units",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/units",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListUnitsRequest) (*apiresource.List[apiresource.Unit], *apierror.APIError) {
			return svc.(UnitSvc).ListUnits
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnit,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
