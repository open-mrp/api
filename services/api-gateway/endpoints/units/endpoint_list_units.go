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
	// Case-insensitive substring match on `name` or `abbreviation`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return units in one of these dimensions.
	Dimensions []constants.UnitDimension `query:"dimensions"`
	// Units in one of these unit groups, as 1.0.forge-preview.9 filtered them.
	UnitGroupIDs []string `query:"unit_group_ids" visibility:"internal"`
}

// Returns a list of units.
//
// Includes the units your account created and the system units every account shares. Units are sorted by creation date, newest first.
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
