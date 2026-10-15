package carrierep

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

// Request to update a carrier.
type UpdateCarrierRequest struct {
	// Carrier ID.
	CarrierID string `path:"id" validate:"required"`
	// Display name, such as `FedEx`.
	//
	// Unique (case-insensitive) among your account's carriers and the system carriers.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Standard Carrier Alpha Code, such as `FDEG`. Send `null` to clear it.
	SCAC field.Clearable[string] `json:"scac,omitzero" validate:"omitempty,min=2,max=4,alpha,uppercase"`
	// Whether customers can see and pick this carrier at checkout in the customer portal.
	//
	// Each of the carrier's service levels has its own `portal_visibility`, which this does not change.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero"`
	// `active` or `archived`. Archived carriers can't be newly set on orders or customer defaults.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Service level selected by default when this carrier is chosen. Send `null` to clear it.
	//
	// Must be one of this carrier's active service levels.
	DefaultServiceLevelID field.Clearable[string] `json:"default_service_level_id,omitzero"`
}

var sampleUpdateCarrierRequest = &UpdateCarrierRequest{
	Name:                  field.Some("FedEx Express"),
	PortalVisibility:      field.Some(constants.CustomerPortalVisibilityVisible),
	DefaultServiceLevelID: field.Set(apiresource.SampleServiceLevelID),
}

func (*UpdateCarrierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateCarrierRequest)
}

// Updates a carrier.
//
// Only fields you send are changed. `type` and `account_number` can't change after creation, and system carriers can't be updated.
type UpdateCarrierEndpoint struct{}

func (e *UpdateCarrierEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateCarrierRequest, *apiresource.Carrier] {
	return (&apiendpoint.APIEndpoint[*UpdateCarrierRequest, *apiresource.Carrier]{
		Title:             "Update Carrier",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/carriers/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCarriers, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeCarrier,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateCarrierRequest) (*apiresource.Carrier, *apierror.APIError) {
			return svc.(CarrierSvc).UpdateCarrier
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCarrier,
			Fields:     []string{"default_service_level", "owner", "owner.account", "service_levels"},
		}),
	})
}
