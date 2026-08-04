package unitgroupep

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

// Parameters for associating a unit with a unit group.
type CreateUnitGroupUnitParam struct {
	// ID of the unit to associate with the group.
	//
	// The unit's dimension must match the group's `type`.
	UnitID string `json:"unit_id" validate:"required"`
	// Share of the unit's price removed when an order is placed in this unit.
	//
	// Expressed as a decimal fraction rather than a whole number, so `0.1` is a 10% discount. Send `0` explicitly for no discount — omitting the field stores a discount of `1`, which removes the entire price.
	DiscountPercentage field.Optional[float64] `json:"discount_percentage,omitzero" default:"1"`
	// Flat amount subtracted from the unit's price when an order is placed in this unit.
	//
	// Subtracted before `discount_percentage` is applied.
	DiscountFixed field.Optional[float64] `json:"discount_fixed,omitzero" default:"0"`
	// Whether the unit is shown to customers in the customer portal.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero" default:"visible"`
}

// Request to create a unit group.
type CreateUnitGroupRequest struct {
	// Display name of the unit group.
	//
	// Must be unique within the account.
	Name string `json:"name" validate:"required,max=255"`
	// Free-form notes about the unit group.
	Notes field.Optional[string] `json:"notes,omitzero" default:"null"`
	// The dimension shared by every unit in this group, such as mass, volume, or currency.
	//
	// The base unit and all associated units must be of this dimension, and the dimension cannot be changed after the group is created.
	Type constants.UnitType `json:"type" validate:"required"`
	// ID of the unit to designate as the group's reference unit.
	//
	// Must be a unit of the group's `type`.
	BaseUnitID string `json:"base_unit_id" validate:"required"`
	// Units to associate with the group, each with its own discount and customer portal visibility.
	AssociatedUnits []CreateUnitGroupUnitParam `json:"associated_units,omitzero"`
}

var sampleCreateUnitGroupDiscountPct = float64(1)
var sampleCreateUnitGroupDiscountFixed = float64(0)
var sampleCreateUnitGroupVisibility = constants.CustomerPortalVisibilityVisible
var sampleCreateUnitGroupNotes = "Used for raw-material weight tracking across the warehouse."
var sampleCreateUnitGroupRequest = &CreateUnitGroupRequest{
	Name:       "Weight Units",
	Notes:      field.Some(sampleCreateUnitGroupNotes),
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

// Creates a unit group, optionally associating units with it in the same request.
//
// The name must be unique within the account, and the base unit and every associated unit must share the group's dimension.
type CreateUnitGroupEndpoint struct{}

func (e *CreateUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUnitGroupRequest, *apiresource.UnitGroup] {
	return (&apiendpoint.APIEndpoint[*CreateUnitGroupRequest, *apiresource.UnitGroup]{
		Title:               "Create Unit Group",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/catalog/unit-groups",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnitGroups, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeUnitGroup,
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
