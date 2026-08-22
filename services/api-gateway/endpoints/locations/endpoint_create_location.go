package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create a location.
type CreateLocationRequest struct {
	// Display name of the location.
	//
	// Maximum 255 characters.
	Name string `json:"name" validate:"required,max=255"`
	// This location's level in the storage hierarchy.
	//
	// The levels run from largest to smallest: `building`, `section`, `aisle`, `rack`, `shelf`, `bin`. They are descriptive labels rather than a rule — the parent you choose is not required to be the next level up.
	TypeCode constants.LocationTypeCode `json:"type" validate:"required"`
	// The location this one sits under in the storage hierarchy.
	//
	// Must be an existing location in your account. Omit to create a top-level location.
	ParentID field.Optional[string] `json:"parent_id,omitzero"`
	// Existing locations to attach beneath the new location.
	//
	// Each listed location is reparented onto the new location, detaching it from its current parent. Every ID must belong to your account.
	ChildIDs field.Optional[[]string] `json:"child_ids,omitzero"`
}

var sampleCreateLocationRequest = &CreateLocationRequest{
	Name:     "Warehouse A",
	TypeCode: "building",
	ParentID: field.Some(apiresource.SampleLocationID),
	ChildIDs: field.Some([]string{apiresource.SampleLocationID}),
}

func (*CreateLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateLocationRequest)
}

// Creates a storage location, optionally placing it in the location hierarchy.
type CreateLocationEndpoint struct{}

func (e *CreateLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateLocationRequest, *apiresource.Location] {
	return (&apiendpoint.APIEndpoint[*CreateLocationRequest, *apiresource.Location]{
		Title:               "Create Location",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/locations",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionCreate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateLocationRequest) (*apiresource.Location, *apierror.APIError) {
			return svc.(LocationSvc).CreateLocation
		},
		LocationFunc: func(resp *apiresource.Location) string {
			return "/v1/operations/locations/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"parent", "children"},
		}),
		ObjectType: constants.ObjectTypeLocation,
	})
}
