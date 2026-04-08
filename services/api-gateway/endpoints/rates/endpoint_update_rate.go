package rateep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateRateRequest is the request to partially update a rate.
type UpdateRateRequest struct {
	// The ID of the rate to update.
	RateID string `path:"id" validate:"required"`
	// The new decimal value of the rate.
	Value *string `json:"value,omitempty"`
	// The new numerator unit ID for this rate.
	NumeratorUnitID *string `json:"numerator_unit_id,omitempty" validate:"omitempty,max=191"`
	// The new denominator unit ID for this rate.
	DenominatorUnitID *string `json:"denominator_unit_id,omitempty" validate:"omitempty,max=191"`
	// The ID of the parent resource that owns this rate.
	ObjectID *string `json:"object_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// The type of the parent resource (e.g. "item", "production_step").
	ObjectType *string `json:"object_type,omitempty" nullable:"false" validate:"omitempty,max=255"`
}

var sampleUpdateRateValue = apiresource.SampleRateValue
var sampleUpdateRateNumeratorUnitID = apiresource.SampleUnitID
var sampleUpdateRateRequest = &UpdateRateRequest{
	Value:           &sampleUpdateRateValue,
	NumeratorUnitID: &sampleUpdateRateNumeratorUnitID,
}

func (*UpdateRateRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateRateRequest)
}

type UpdateRateEndpoint struct{}

func (e *UpdateRateEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateRateRequest, *apiresource.Rate] {
	return &apiendpoint.APIEndpoint[*UpdateRateRequest, *apiresource.Rate]{
		Title:             "Update Rate",
		Description:       "Partially updates a rate record.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/rates/{id}",
		ContentType:       "application/json",
		Request:           &UpdateRateRequest{},
		Response:          &apiresource.Rate{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateRateRequest) (*apiresource.Rate, *apierror.APIError) {
			return svc.(RateSvc).UpdateRate
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRate,
			Fields:     []string{"numerator_unit", "denominator_unit"},
		}),
	}
}
