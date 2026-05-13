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

// CreateUnitGroupUnitParam contains parameters for an associated unit.
type CreateUnitGroupUnitParam struct {
	// Unit ID.
	UnitID string `json:"unit_id" validate:"required"`
	// Discount percentage.
	DiscountPercentage *float64 `json:"discount_percentage,omitempty" default:"1" nullable:"false"`
	// Fixed discount amount.
	DiscountFixed *float64 `json:"discount_fixed,omitempty" default:"0" nullable:"false"`
	// Customer portal visibility.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility,omitempty" default:"visible" nullable:"false"`
}

// CreateUnitGroupRequest is a request to create a unit group.
type CreateUnitGroupRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Notes.
	Notes *string `json:"notes,omitempty" default:"null" nullable:"false"`
	// Unit type.
	Type constants.UnitType `json:"type" validate:"required"`
	// Base unit ID.
	BaseUnitID string `json:"base_unit_id" validate:"required"`
	// Associated units to create with the group.
	AssociatedUnits []CreateUnitGroupUnitParam `json:"associated_units,omitempty"`
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
			DiscountPercentage:       &sampleCreateUnitGroupDiscountPct,
			DiscountFixed:            &sampleCreateUnitGroupDiscountFixed,
			CustomerPortalVisibility: &sampleCreateUnitGroupVisibility,
		},
	},
}

func (*CreateUnitGroupRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateUnitGroupRequest)
}

type CreateUnitGroupEndpoint struct{}

func (e *CreateUnitGroupEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUnitGroupRequest, *apiresource.UnitGroup] {
	return &apiendpoint.APIEndpoint[*CreateUnitGroupRequest, *apiresource.UnitGroup]{
		Title:             "Create Unit Group",
		Description:       "Creates a unit group with optional associated units.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/unit-groups",
		Request:           &CreateUnitGroupRequest{},
		Response:          &apiresource.UnitGroup{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
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
	}
}
