package demandoverridesep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list demand overrides.
type ListDemandOverridesRequest struct {
	apiresource.PaginationRequest
	// Only return overrides targeting these kinds of resource.
	ScopeTypes []constants.DemandOverrideScope `query:"scope_types"`
	// Only return overrides targeting these items or product lines.
	ScopeRefIDs []string `query:"scope_ref_ids"`
	// Only return overrides making these kinds of adjustment.
	Adjustments []constants.DemandOverrideAdjustment `query:"adjustments"`
	// Only return overrides in these activation states.
	Statuses []constants.ActivationStatus `query:"statuses"`
	// Only return overrides whose period ends on or after this timestamp, formatted as RFC3339.
	PeriodStart *string `query:"period_start"`
	// Only return overrides whose period starts on or before this timestamp, formatted as RFC3339.
	PeriodEnd *string `query:"period_end"`
}

// Returns a paginated list of demand overrides, most recently created first.
//
// The period filters match on overlap rather than containment, so an override spanning a quarter is returned when querying a single month inside it.
type ListDemandOverridesEndpoint struct{}

func (e *ListDemandOverridesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListDemandOverridesRequest, *apiresource.List[apiresource.DemandOverride]] {
	return (&apiendpoint.APIEndpoint[*ListDemandOverridesRequest, *apiresource.List[apiresource.DemandOverride]]{
		Title:             "List Demand Overrides",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/demand-overrides",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeDemandOverride,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDemandOverrides, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListDemandOverridesRequest) (*apiresource.List[apiresource.DemandOverride], *apierror.APIError) {
			return svc.(DemandOverridesSvc).ListDemandOverrides
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeDemandOverride,
			Fields:     []string{"scope"},
		}),
	})
}
