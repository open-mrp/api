package unitep

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

// Request to partially update a unit.
type UpdateUnitRequest struct {
	// Unit ID.
	UnitID string `path:"id" validate:"required"`
	// Display name of the unit.
	//
	// Must be unique within the account.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Short abbreviation for the unit.
	//
	// Must be unique within the account.
	Abbreviation field.Optional[string] `json:"abbreviation,omitzero" validate:"omitempty"`
	// Numerator of the ratio that converts a quantity in this unit into the dimension's base unit.
	//
	// A quantity is converted with `value × (ratio_numerator / ratio_denominator) + (offset_numerator / offset_denominator)`.
	RatioNumerator field.Optional[string] `json:"ratio_numerator,omitzero" validate:"omitempty,decimal" format:"decimal"`
	// Denominator of the ratio that converts a quantity in this unit into the dimension's base unit.
	//
	// Must not be zero.
	RatioDenominator field.Optional[string] `json:"ratio_denominator,omitzero" validate:"omitempty,nonzero_decimal" format:"decimal"`
	// Numerator of the conversion offset, applied after the ratio for scales that do not share a zero point, such as temperature.
	OffsetNumerator field.Optional[string] `json:"offset_numerator,omitzero" validate:"omitempty,decimal" format:"decimal"`
	// Denominator of the conversion offset.
	//
	// Must not be zero.
	OffsetDenominator field.Optional[string] `json:"offset_denominator,omitzero" validate:"omitempty,nonzero_decimal" format:"decimal"`
}

var sampleUpdateUnitRequest = &UpdateUnitRequest{
	Name:              field.Some("Kilogram"),
	Abbreviation:      field.Some("kg"),
	RatioNumerator:    field.Some("1000"),
	RatioDenominator:  field.Some("1"),
	OffsetNumerator:   field.Some("0"),
	OffsetDenominator: field.Some("1"),
}

func (*UpdateUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitRequest)
}

// Partially updates a unit owned by your account.
//
// System units cannot be modified, and a unit's dimension is fixed once it is created.
type UpdateUnitEndpoint struct{}

func (e *UpdateUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitRequest, *apiresource.Unit] {
	return (&apiendpoint.APIEndpoint[*UpdateUnitRequest, *apiresource.Unit]{
		Title:               "Update Unit",
		Method:              http.MethodPatch,
		Route:               "/v1/catalog/units/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUnitRequest) (*apiresource.Unit, *apierror.APIError) {
			return svc.(UnitSvc).UpdateUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnit,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
