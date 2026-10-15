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

// Request to create a service level.
type CreateServiceLevelRequest struct {
	// The carrier that will offer this service level.
	CarrierID string `path:"carrier_id" validate:"required"`
	// Display name, shown to customers at checkout when the service level is visible.
	Name string `json:"name" validate:"required,max=255"`
	// Code identifying the service, such as `fedex_ground`.
	//
	// Unique among the carrier's service levels. For connected carriers it's the carrier's own code, used for rates and labels.
	Code string `json:"code" validate:"required,max=255"`
	// Whether customers can see and pick this service level at checkout in the customer portal. Defaults to `visible`.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero" default:"visible"`
	// `active` or `archived`. Defaults to `active`. Archived service levels can't be newly set on orders or customer defaults.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Business days this service usually takes, used when the carrier can't quote a lane.
	//
	// Works an order's ship-by date back from its promised date. Omit when unknown.
	EstimatedTransitDays field.Optional[int32] `json:"estimated_transit_days,omitzero" validate:"omitempty,gte=0,lte=365"`
	// Makes this service level its carrier's default. Kept for 1.0.forge-preview.9 callers; set the carrier's `default_service_level_id` instead.
	IsDefault field.Optional[bool] `json:"is_default,omitzero" visibility:"internal"`
}

var sampleCreateServiceLevelRequest = &CreateServiceLevelRequest{
	Name:                 "Ground Shipping",
	Code:                 "ground",
	PortalVisibility:     field.Some(constants.CustomerPortalVisibilityVisible),
	EstimatedTransitDays: field.Some(int32(3)),
}

func (*CreateServiceLevelRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateServiceLevelRequest)
}

// Adds a service level to a carrier.
//
// Use it for carriers you manage yourself, or for a service a connected carrier doesn't publish. Service levels you add are never archived by a sync.
type CreateServiceLevelEndpoint struct{}

func (e *CreateServiceLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateServiceLevelRequest, *apiresource.ServiceLevel] {
	return (&apiendpoint.APIEndpoint[*CreateServiceLevelRequest, *apiresource.ServiceLevel]{
		Title:             "Create Service Level",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{carrier_id}/service-levels",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeServiceLevel,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateServiceLevelRequest) (*apiresource.ServiceLevel, *apierror.APIError) {
			return svc.(ServiceLevelSvc).CreateServiceLevel
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeServiceLevel,
			Fields:     []string{"carrier", "owner", "owner.account"},
		}),
	})
}
