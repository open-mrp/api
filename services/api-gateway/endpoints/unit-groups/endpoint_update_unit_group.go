package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateUnitGroupRequest is the request to partially update a unit group.
type UpdateUnitGroupRequest struct {
	// The ID of the unit group to update.
	UnitGroupID string `path:"id" validate:"required"`
	// The display name of the unit group.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Optional notes about the unit group. Set to null to clear.
	Notes *string `json:"notes,omitempty" nullable:"true"`
	// The base unit ID.
	BaseUnitID *string `json:"base_unit_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// If provided, upserts associated units. Existing associated units not in the list are preserved.
	AssociatedUnits *[]CreateUnitGroupUnitParam `json:"associated_units,omitempty" nullable:"false"`
}

var sampleUpdateUnitGroupName = "Weight Units (Updated)"
var sampleUpdateUnitGroupBaseUnitID = apiresource.SampleUnitID
var sampleUpdateUnitGroupRequest = &UpdateUnitGroupRequest{
	Name:       &sampleUpdateUnitGroupName,
	BaseUnitID: &sampleUpdateUnitGroupBaseUnitID,
}

func (*UpdateUnitGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitGroupRequest)
}

type UpdateUnitGroupEndpoint struct{}

func (e *UpdateUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitGroupRequest, *apiresource.UnitGroup] {
	return &apiendpoint.APIEndpoint[*UpdateUnitGroupRequest, *apiresource.UnitGroup]{
		Title:             "Update Unit Group",
		Description:       "Partially updates a unit group. System unit groups cannot be updated.",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/unit-groups/{id}",
		ContentType:       "application/json",
		Request:           &UpdateUnitGroupRequest{},
		Response:          &apiresource.UnitGroup{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
			return svc.(UnitGroupSvc).UpdateUnitGroup
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroup,
			Fields:     []string{"owner", "owner.account", "base_unit", "associated_units"},
		}),
	}
}
