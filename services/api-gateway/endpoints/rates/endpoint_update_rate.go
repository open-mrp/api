package rateep

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

// Request to partially update a rate.
type UpdateRateRequest struct {
	// Rate ID.
	RateID string `path:"id" validate:"required"`
	// Decimal value of the rate.
	Value field.Optional[string] `json:"value,omitzero"`
	// Numerator unit ID.
	NumeratorUnitID field.Optional[string] `json:"numerator_unit_id,omitzero" validate:"omitempty"`
	// Denominator unit ID.
	DenominatorUnitID field.Optional[string] `json:"denominator_unit_id,omitzero" validate:"omitempty"`
	// Parent resource ID.
	ObjectID field.Optional[string] `json:"object_id,omitzero" validate:"omitempty"`
	// Parent resource type (e.g. "item", "production_step").
	ObjectType field.Optional[string] `json:"object_type,omitzero" validate:"omitempty,max=255"`
}

var sampleUpdateRateValue = apiresource.SampleRateValue
var sampleUpdateRateNumeratorUnitID = apiresource.SampleUnitID
var sampleUpdateRateRequest = &UpdateRateRequest{
	Value:           field.Some(sampleUpdateRateValue),
	NumeratorUnitID: field.Some(sampleUpdateRateNumeratorUnitID),
}

func (*UpdateRateRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateRateRequest)
}

// Partially updates a rate.
type UpdateRateEndpoint struct{}

func (e *UpdateRateEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateRateRequest, *apiresource.Rate] {
	return (&apiendpoint.APIEndpoint[*UpdateRateRequest, *apiresource.Rate]{
		Title:             "Update Rate",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/rates/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeRate,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateRateRequest) (*apiresource.Rate, *apierror.APIError) {
			return svc.(RateSvc).UpdateRate
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRate,
			Fields:     []string{"numerator_unit", "denominator_unit"},
		}),
	})
}
