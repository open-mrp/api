package servicelevelep

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

// Request to update a service level.
type UpdateServiceLevelRequest struct {
	// The carrier that owns this service level.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `path:"id" validate:"required"`
	// Human-readable name for the service level, shown to customers at checkout when the service level is visible.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Carrier-specific code identifying this service level (e.g. `fedex_ground`).
	//
	// Must be unique among the carrier's service levels. For a service level synced from a connected carrier the `service_level_token` used for rating is fixed by the carrier and a code change does not affect it; for one you created yourself, the token follows the code.
	Code field.Optional[string] `json:"code,omitzero" validate:"omitempty,max=255"`
	// Whether customers can see and select this service level at checkout in the customer portal.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero"`
	// Whether this is the carrier's default service level, pre-selected when the carrier is chosen.
	//
	// Each carrier has at most one default; setting this to `true` clears the carrier's existing default.
	IsDefault field.Optional[bool] `json:"is_default,omitzero"`
	// Business days this service typically takes in transit, used to work an order's ship-by date back from a promised delivery date.
	//
	// A fallback: when a carrier can rate the lane, the transit it quotes is used instead. Set to null to remove it, which leaves transit unknown for lanes the carrier cannot rate.
	DefaultTransitDays field.Clearable[int32] `json:"default_transit_days,omitzero" validate:"omitempty,gte=0,lte=365"`
}

var sampleUpdateServiceLevelName = "Express Shipping"
var sampleUpdateServiceLevelRequest = &UpdateServiceLevelRequest{
	Name:                     field.Some(sampleUpdateServiceLevelName),
	Code:                     field.Some("express"),
	CustomerPortalVisibility: field.Some(constants.CustomerPortalVisibilityVisible),
}

func (*UpdateServiceLevelRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateServiceLevelRequest)
}

// Updates a service level's name, code, customer portal visibility, or default status.
//
// Only the fields you send are changed. System-owned service levels cannot be updated.
type UpdateServiceLevelEndpoint struct{}

func (e *UpdateServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateServiceLevelRequest, *apiresource.ServiceLevel] {
	return (&apiendpoint.APIEndpoint[*UpdateServiceLevelRequest, *apiresource.ServiceLevel]{
		Title:               "Update Service Level",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeServiceLevel,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).UpdateServiceLevel
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
