package rateep

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

// Request to partially update a rate.
type UpdateRateRequest struct {
	// Rate ID.
	RateID string `path:"id" validate:"required"`
	// New decimal value for the rate, expressed as the amount of the numerator unit per one denominator unit.
	Value field.Optional[string] `json:"value,omitzero"`
	// ID of the new unit for the rate's numerator (e.g. the currency of a price).
	//
	// The stored value is kept as-is and is not converted into the new unit, so send `value` alongside this when the amount should change too.
	NumeratorUnitID field.Optional[string] `json:"numerator_unit_id,omitzero" validate:"omitempty"`
	// ID of the new unit for the rate's denominator (the per-unit basis).
	//
	// As with the numerator, the value is not re-scaled when the unit changes.
	DenominatorUnitID field.Optional[string] `json:"denominator_unit_id,omitzero" validate:"omitempty"`
	// ID of the resource that owns this rate.
	//
	// Used together with `object_type` to verify the owning resource exists; it does not reassign the rate.
	ObjectID field.Optional[string] `json:"object_id,omitzero" validate:"omitempty"`
	// Type of the resource that owns this rate.
	//
	// Determines the permission required for the update. Must be `item`, `production_step`, or `department`.
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

// Updates the value or units of a rate in place.
//
// A rate belongs to the resource that reports it — an item's unit price or cost, a department's labor rate, and so on — so this changes that resource's stored rate directly.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainItems, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainProductionSteps, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeRate,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateRateRequest) (*apiresource.Rate, *apierror.APIError) {
			return svc.(RateSvc).UpdateRate
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeRate,
			Fields:     []string{"numerator_unit", "denominator_unit"},
		}),
	})
}
