package unitep

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

// Request to update a unit.
type UpdateUnitRequest struct {
	// Unit ID.
	UnitID string `path:"id" validate:"required"`
	// Display name, such as `Kilogram`.
	//
	// Unique (case-insensitive) among your account's units and the system units.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Short label, such as `kg`.
	//
	// Unique (case-insensitive) among your account's units and the system units.
	Abbreviation field.Optional[string] `json:"abbreviation,omitzero" validate:"omitempty,max=32"`
	// The ratio that converts a value in this unit to its dimension's base unit.
	//
	// A value converts as `value × ratio_numerator ÷ ratio_denominator + offset_numerator ÷ offset_denominator`.
	RatioNumerator field.Optional[string] `json:"ratio_numerator,omitzero" validate:"omitempty,decimal" format:"decimal"`
	// The ratio that converts a value in this unit to its dimension's base unit.
	//
	// The denominator can't be zero.
	RatioDenominator field.Optional[string] `json:"ratio_denominator,omitzero" validate:"omitempty,nonzero_decimal" format:"decimal"`
	// An offset added after the ratio, for scales with different zero points, such as temperature.
	OffsetNumerator field.Optional[string] `json:"offset_numerator,omitzero" validate:"omitempty,decimal" format:"decimal"`
	// An offset added after the ratio, for scales with different zero points, such as temperature.
	//
	// The denominator can't be zero.
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

// Updates a unit.
//
// Only fields you send are changed. System units can't be updated, and `dimension` can't change after creation.
type UpdateUnitEndpoint struct{}

func (e *UpdateUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitRequest, *apiresource.Unit] {
	return (&apiendpoint.APIEndpoint[*UpdateUnitRequest, *apiresource.Unit]{
		Title:             "Update Unit",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/units/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUnitRequest) (*apiresource.Unit, *apierror.APIError) {
			return svc.(UnitSvc).UpdateUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnit,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
