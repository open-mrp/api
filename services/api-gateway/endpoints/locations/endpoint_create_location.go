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

// Request to create a location.
type CreateLocationRequest struct {
	// Display name, unique (case-insensitive) among your account's locations.
	//
	// Maximum 255 characters.
	Name string `json:"name" validate:"required,max=255"`
	// What kind of place this is.
	//
	// `site` is a plant, warehouse or 3PL: always top-level, and the only type with an `address`. The rest run from largest to smallest: `building`, `section`, `aisle`, `rack`, `shelf`, `bin`. They are descriptive labels rather than a rule — the parent you choose is not required to be the next level up.
	TypeCode constants.LocationTypeCode `json:"type" validate:"required"`
	// `active` (the default) or `archived`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Where the site is. Only a `site` takes an address.
	//
	// Must be one of your account's addresses.
	AddressID field.Optional[string] `json:"address_id,omitzero"`
	// The location this one sits under.
	//
	// Must be one of your account's active locations. Omit to create a top-level location; a site is always top-level.
	ParentID field.Optional[string] `json:"parent_id,omitzero"`
	// Existing locations to place beneath the new location.
	//
	// Each listed location is moved onto the new location, detaching it from its current parent. A site can't be listed.
	ChildIDs field.Optional[[]string] `json:"child_ids,omitzero"`
}

var sampleCreateLocationRequest = &CreateLocationRequest{
	Name:     "Warehouse A",
	TypeCode: constants.LocationTypeCodeBuilding,
	ParentID: field.Some(apiresource.SampleLocationID),
	ChildIDs: field.Some([]string{apiresource.SampleLocationChildID}),
}

func (*CreateLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateLocationRequest)
}

// Creates a location.
//
// Create a `site` for each plant, warehouse or 3PL, then the buildings, aisles and bins inside it. Your account's first site becomes its default site.
type CreateLocationEndpoint struct{}

func (e *CreateLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateLocationRequest, *apiresource.Location] {
	return (&apiendpoint.APIEndpoint[*CreateLocationRequest, *apiresource.Location]{
		Title:             "Create Location",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionCreate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateLocationRequest) (*apiresource.Location, *apierror.APIError) {
			return svc.(LocationSvc).CreateLocation
		},
		LocationFunc: func(resp *apiresource.Location) string {
			return "/v1/operations/locations/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"address", "parent", "children"},
		}),
		ObjectType: constants.ObjectTypeLocation,
	})
}
