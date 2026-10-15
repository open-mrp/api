package locationep

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

// Request to update a location.
type UpdateLocationRequest struct {
	// Location ID.
	LocationID string `path:"id" validate:"required"`
	// Display name, unique (case-insensitive) among your account's locations.
	//
	// Maximum 255 characters.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// What kind of place this is.
	//
	// `site` is a plant, warehouse or 3PL: always top-level, and the only type with an `address`. The rest run from largest to smallest: `building`, `section`, `aisle`, `rack`, `shelf`, `bin`. Your default site stays a `site`.
	TypeCode field.Optional[constants.LocationTypeCode] `json:"type,omitzero"`
	// `active` or `archived`. Your default site can't be archived.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Where the site is. Only a `site` takes an address.
	//
	// Must be one of your account's addresses. Send `null` to remove it.
	AddressID field.Clearable[string] `json:"address_id,omitzero"`
	// The location this one sits under.
	//
	// Must be one of your account's active locations, and not this location or one below it. Send `null` to make it a top-level location.
	ParentID field.Clearable[string] `json:"parent_id,omitzero" validate:"omitempty"`
	// The locations that sit directly beneath this one.
	//
	// Replaces the full set of children: current children not listed become top-level locations, and listed locations are moved onto this one. Send `null` to detach every child; omit to leave them as they are. A site can't be listed.
	ChildIDs field.Clearable[[]string] `json:"child_ids,omitzero"`
}

var sampleUpdateLocationRequest = &UpdateLocationRequest{
	Name:     field.Some("Warehouse B"),
	TypeCode: field.Some(constants.LocationTypeCodeSection),
	ParentID: field.Set(apiresource.SampleLocationID),
	ChildIDs: field.Set([]string{apiresource.SampleLocationChildID}),
}

func (*UpdateLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateLocationRequest)
}

// Updates a location.
//
// Only fields you send are changed. Archiving keeps the location's stock, history and children; it just can't be chosen for new stock or as a new parent.
type UpdateLocationEndpoint struct{}

func (e *UpdateLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateLocationRequest, *apiresource.Location] {
	return (&apiendpoint.APIEndpoint[*UpdateLocationRequest, *apiresource.Location]{
		Title:             "Update Location",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateLocationRequest) (*apiresource.Location, *apierror.APIError) {
			return svc.(LocationSvc).UpdateLocation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"address", "parent", "children"},
		}),
		ObjectType: constants.ObjectTypeLocation,
	})
}
