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

// UpdateUnitGroupUnitRequest is the request to update an associated unit.
type UpdateUnitGroupUnitRequest struct {
	// The ID of the unit group.
	UnitGroupID string `path:"unitGroupId" validate:"required"`
	// The ID of the associated unit.
	AssociatedUnitID string `path:"id" validate:"required"`
	// The unit ID.
	UnitID *string `json:"unit_id,omitempty"`
	// The discount percentage.
	DiscountPercentage *float64 `json:"discount_percentage,omitempty"`
	// The fixed discount amount.
	DiscountFixed *float64 `json:"discount_fixed,omitempty"`
	// Whether this associated unit is visible in the customer portal.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility,omitempty" nullable:"false"`
}

var sampleUpdateUnitGroupUnitDiscountPct = float64(0.9)
var sampleUpdateUnitGroupUnitUnitID = apiresource.SampleUnitID
var sampleUpdateUnitGroupUnitRequest = &UpdateUnitGroupUnitRequest{
	UnitID:             &sampleUpdateUnitGroupUnitUnitID,
	DiscountPercentage: &sampleUpdateUnitGroupUnitDiscountPct,
}

func (*UpdateUnitGroupUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitGroupUnitRequest)
}

type UpdateUnitGroupUnitEndpoint struct{}

func (e *UpdateUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitGroupUnitRequest, *apiresource.UnitGroupUnit] {
	return &apiendpoint.APIEndpoint[*UpdateUnitGroupUnitRequest, *apiresource.UnitGroupUnit]{
		Title:             "Update Unit Group Associated Unit",
		Description:       "Updates an associated unit within a unit group.",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/unit-groups/{unitGroupId}/units/{id}",
		ContentType:       "application/json",
		Request:           &UpdateUnitGroupUnitRequest{},
		Response:          &apiresource.UnitGroupUnit{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
			return svc.(UnitGroupSvc).UpdateUnitGroupUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroupUnit,
			Fields:     []string{"unit"},
		}),
	}
}
