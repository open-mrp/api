package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update a unit.
type UpdateUnitRequest struct {
	// Unit ID.
	UnitID string `path:"id" validate:"required"`
	// Display name of the unit.
	Name *string `json:"name,omitempty" validate:"omitempty,max=255"`
	// Short abbreviation for the unit.
	Abbreviation *string `json:"abbreviation,omitempty" validate:"omitempty"`
	// Conversion ratio numerator, as a decimal string.
	RatioNumerator *string `json:"ratio_numerator,omitempty" format:"decimal"`
	// Conversion ratio denominator, as a decimal string. Must not be zero.
	RatioDenominator *string `json:"ratio_denominator,omitempty" validate:"omitempty,nonzero_decimal" format:"decimal"`
	// Conversion offset numerator, as a decimal string.
	OffsetNumerator *string `json:"offset_numerator,omitempty" format:"decimal"`
	// Conversion offset denominator, as a decimal string. Must not be zero.
	OffsetDenominator *string `json:"offset_denominator,omitempty" validate:"omitempty,nonzero_decimal" format:"decimal"`
}

var sampleUpdateUnitRequest = &UpdateUnitRequest{
	Name:         new("Kilogram"),
	Abbreviation: new("kg"),
}

func (*UpdateUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitRequest)
}

// Partially updates an account-owned unit; system units cannot be updated.
type UpdateUnitEndpoint struct{}

func (e *UpdateUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitRequest, *apiresource.Unit] {
	return (&apiendpoint.APIEndpoint[*UpdateUnitRequest, *apiresource.Unit]{
		Title:             "Update Unit",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/units/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
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
