package unitgroupep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// CreateUnitGroupUnitParam contains parameters for an associated unit.
type CreateUnitGroupUnitParam struct {
	// Unit ID.
	UnitID string `json:"unit_id" validate:"required"`
	// Discount percentage.
	DiscountPercentage field.Optional[float64] `json:"discount_percentage,omitzero" default:"1"`
	// Fixed discount amount.
	DiscountFixed field.Optional[float64] `json:"discount_fixed,omitzero" default:"0"`
	// Customer portal visibility.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero" default:"visible"`
}

// CreateUnitGroupRequest is a request to create a unit group.
type CreateUnitGroupRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Notes.
	Notes field.Optional[string] `json:"notes,omitzero" default:"null"`
	// Unit type.
	Type constants.UnitType `json:"type" validate:"required"`
	// Base unit ID.
	BaseUnitID string `json:"base_unit_id" validate:"required"`
	// Associated units to create with the group.
	AssociatedUnits []CreateUnitGroupUnitParam `json:"associated_units,omitzero"`
}

var sampleCreateUnitGroupDiscountPct = float64(1)
var sampleCreateUnitGroupDiscountFixed = float64(0)
var sampleCreateUnitGroupVisibility = constants.CustomerPortalVisibilityVisible
var sampleCreateUnitGroupRequest = &CreateUnitGroupRequest{
	Name:       "Weight Units",
	Type:       constants.UnitTypeMass,
	BaseUnitID: apiresource.SampleUnitID,
	AssociatedUnits: []CreateUnitGroupUnitParam{
		{
			UnitID:                   apiresource.SampleUnitID,
			DiscountPercentage:       field.Some(sampleCreateUnitGroupDiscountPct),
			DiscountFixed:            field.Some(sampleCreateUnitGroupDiscountFixed),
			CustomerPortalVisibility: field.Some(sampleCreateUnitGroupVisibility),
		},
	},
}

func (*CreateUnitGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateUnitGroupRequest)
}

// Creates a unit group with optional associated units.
type CreateUnitGroupEndpoint struct{}

func (e *CreateUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUnitGroupRequest, *apiresource.UnitGroup] {
	return (&apiendpoint.APIEndpoint[*CreateUnitGroupRequest, *apiresource.UnitGroup]{
		Title:             "Create Unit Group",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/unit-groups",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeUnitGroup,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateUnitGroupRequest) (*apiresource.UnitGroup, *apierror.APIError) {
			return svc.(UnitGroupSvc).CreateUnitGroup
		},
		LocationFunc: func(resp *apiresource.UnitGroup) string {
			return "/v1/catalog/unit-groups/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroup,
			Fields:     []string{"owner", "owner.account", "base_unit", "associated_units"},
		}),
	})
}
