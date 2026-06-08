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

// UpdateUnitGroupUnitRequest is a request to update an associated unit.
type UpdateUnitGroupUnitRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
	// Unit group unit ID.
	AssociatedUnitID string `path:"id" validate:"required"`
	// Unit ID.
	UnitID field.Optional[string] `json:"unit_id,omitzero" validate:"omitempty"`
	// Discount percentage.
	DiscountPercentage field.Optional[float64] `json:"discount_percentage,omitzero"`
	// Fixed discount amount.
	DiscountFixed field.Optional[float64] `json:"discount_fixed,omitzero"`
	// Customer portal visibility.
	CustomerPortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"customer_portal_visibility,omitzero"`
}

var sampleUpdateUnitGroupUnitDiscountPct = float64(0.9)
var sampleUpdateUnitGroupUnitUnitID = apiresource.SampleUnitID
var sampleUpdateUnitGroupUnitRequest = &UpdateUnitGroupUnitRequest{
	UnitID:             field.Some(sampleUpdateUnitGroupUnitUnitID),
	DiscountPercentage: field.Some(sampleUpdateUnitGroupUnitDiscountPct),
}

func (*UpdateUnitGroupUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitGroupUnitRequest)
}

// Partially updates an associated unit within a unit group.
type UpdateUnitGroupUnitEndpoint struct{}

func (e *UpdateUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitGroupUnitRequest, *apiresource.UnitGroupUnit] {
	return (&apiendpoint.APIEndpoint[*UpdateUnitGroupUnitRequest, *apiresource.UnitGroupUnit]{
		Title:             "Update Unit Group Associated Unit",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/unit-groups/{unit_group_id}/units/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeUnitGroupUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
			return svc.(UnitGroupSvc).UpdateUnitGroupUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroupUnit,
			Fields:     []string{"unit"},
		}),
	})
}
