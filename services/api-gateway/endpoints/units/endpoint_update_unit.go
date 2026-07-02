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
	// Conversion ratio numerator relative to the base unit.
	RatioNumerator field.Optional[string] `json:"ratio_numerator,omitzero" validate:"omitempty,decimal" format:"decimal"`
	// Conversion ratio denominator relative to the base unit.
	//
	// Must not be zero.
	RatioDenominator field.Optional[string] `json:"ratio_denominator,omitzero" validate:"omitempty,nonzero_decimal" format:"decimal"`
	// Conversion offset numerator, used for temperature-like conversions.
	OffsetNumerator field.Optional[string] `json:"offset_numerator,omitzero" validate:"omitempty,decimal" format:"decimal"`
	// Conversion offset denominator.
	//
	// Must not be zero.
	OffsetDenominator field.Optional[string] `json:"offset_denominator,omitzero" validate:"omitempty,nonzero_decimal" format:"decimal"`
}

var sampleUpdateUnitRequest = &UpdateUnitRequest{
	Name:         field.Some("Kilogram"),
	Abbreviation: field.Some("kg"),
}

func (*UpdateUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitRequest)
}

// Partially updates an account-owned unit; system units cannot be updated.
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
