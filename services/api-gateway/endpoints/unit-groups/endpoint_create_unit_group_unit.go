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

// Request to add a unit to a unit group.
type CreateUnitGroupUnitRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
	// ID of the unit to associate with the group.
	//
	// The unit's dimension must match the group's `type`.
	UnitID string `json:"unit_id" validate:"required"`
	// Percentage discount applied to the unit's price when an order is placed in this unit (e.g. `10` is a 10% discount).
	DiscountPercentage field.Optional[float64] `json:"discount_percentage,omitzero" default:"1"`
	// Flat amount subtracted from the unit's price when an order is placed in this unit.
	DiscountFixed field.Optional[float64] `json:"discount_fixed,omitzero" default:"0"`
	// Whether the unit is shown to customers in the customer portal.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero" default:"visible"`
}

var sampleCreateUnitGroupUnitDiscountPct = float64(1)
var sampleCreateUnitGroupUnitDiscountFixed = float64(0)
var sampleCreateUnitGroupUnitVisibility = constants.CustomerPortalVisibilityVisible
var sampleCreateUnitGroupUnitRequest = &CreateUnitGroupUnitRequest{
	UnitID:                   apiresource.SampleUnitID,
	DiscountPercentage:       field.Some(sampleCreateUnitGroupUnitDiscountPct),
	DiscountFixed:            field.Some(sampleCreateUnitGroupUnitDiscountFixed),
	CustomerPortalVisibility: field.Some(sampleCreateUnitGroupUnitVisibility),
}

func (*CreateUnitGroupUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateUnitGroupUnitRequest)
}

// Adds a unit to a unit group. If the unit is already in the group, its existing association is updated with the provided settings instead.
type CreateUnitGroupUnitEndpoint struct{}

func (e *CreateUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUnitGroupUnitRequest, *apiresource.UnitGroupUnit] {
	return (&apiendpoint.APIEndpoint[*CreateUnitGroupUnitRequest, *apiresource.UnitGroupUnit]{
		Title:             "Create Unit Group Associated Unit",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/unit-groups/{unit_group_id}/units",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeUnitGroupUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
			return svc.(UnitGroupSvc).CreateUnitGroupUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroupUnit,
			Fields:     []string{"unit"},
		}),
	})
}
