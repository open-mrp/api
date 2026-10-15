package servicelevelep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update a service level.
type UpdateServiceLevelRequest struct {
	// The carrier that owns this service level.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Service level ID.
	ServiceLevelID string `path:"id" validate:"required"`
	// Display name, shown to customers at checkout when the service level is visible.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Code identifying the service, such as `fedex_ground`.
	//
	// Unique among the carrier's service levels. For connected carriers it's the carrier's own code, used for rates and labels.
	Code field.Optional[string] `json:"code,omitzero" validate:"omitempty,max=255"`
	// Whether customers can see and pick this service level at checkout in the customer portal.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero"`
	// `active` or `archived`. Archived service levels can't be newly set on orders or customer defaults, and a carrier's default can't be archived.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Business days this service usually takes, used when the carrier can't quote a lane. Send `null` when unknown.
	EstimatedTransitDays field.Clearable[int32] `json:"estimated_transit_days,omitzero" validate:"omitempty,gte=0,lte=365"`
	// `true` makes this service level its carrier's default; `false` stops it being the default. Kept for 1.0.forge-preview.9 callers; set the carrier's `default_service_level_id` instead.
	IsDefault field.Optional[bool] `json:"is_default,omitzero" visibility:"internal"`
}

var sampleUpdateServiceLevelRequest = &UpdateServiceLevelRequest{
	Name:             field.Some("Express Shipping"),
	Code:             field.Some("express"),
	PortalVisibility: field.Some(constants.CustomerPortalVisibilityVisible),
}

func (*UpdateServiceLevelRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateServiceLevelRequest)
}

// Updates a service level.
//
// Only fields you send are changed. System-owned service levels can't be updated.
type UpdateServiceLevelEndpoint struct{}

func (e *UpdateServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateServiceLevelRequest, *apiresource.ServiceLevel] {
	return (&apiendpoint.APIEndpoint[*UpdateServiceLevelRequest, *apiresource.ServiceLevel]{
		Title:             "Update Service Level",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeServiceLevel,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).UpdateServiceLevel
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"carrier", "owner", "owner.account"},
		}),
	})
}
