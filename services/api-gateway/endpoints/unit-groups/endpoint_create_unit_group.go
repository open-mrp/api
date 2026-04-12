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

// CreateUnitGroupUnitParam carries data for a single associated unit.
type CreateUnitGroupUnitParam struct {
	// The unit ID.
	UnitID string `json:"unit_id" validate:"required,max=191"`
	// The discount percentage.
	DiscountPercentage *float64 `json:"discount_percentage,omitempty" default:"1" nullable:"false"`
	// The fixed discount amount.
	DiscountFixed *float64 `json:"discount_fixed,omitempty" default:"0" nullable:"false"`
	// Whether this associated unit is visible in the customer portal.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility,omitempty" default:"visible" nullable:"false"`
}

// CreateUnitGroupRequest is the request to create a new unit group.
type CreateUnitGroupRequest struct {
	// The display name of the unit group.
	Name string `json:"name" validate:"required,max=255"`
	// Optional notes about the unit group.
	Notes *string `json:"notes,omitempty" default:"null"`
	// The unit type code (e.g. "mass", "quantity").
	Type constants.UnitType `json:"type" validate:"required"`
	// The base unit ID.
	BaseUnitID string `json:"base_unit_id" validate:"required,max=191"`
	// Optional associated units to create with the group.
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
		Description:       "Creates a new unit group with optional associated units.",
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
			Fields:     []string{"owner", "base_unit", "associated_units"},
		}),
	}
}
