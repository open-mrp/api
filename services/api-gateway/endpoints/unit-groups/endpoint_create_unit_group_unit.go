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

// CreateUnitGroupUnitRequest is a request to create an associated unit within a unit group.
type CreateUnitGroupUnitRequest struct {
	// Unit group ID.
	UnitGroupID string `path:"unit_group_id" validate:"required"`
	// Unit ID.
	UnitID string `json:"unit_id" validate:"required"`
	// Discount percentage.
	DiscountPercentage *float64 `json:"discount_percentage,omitempty" default:"1" nullable:"false"`
	// Fixed discount amount.
	DiscountFixed *float64 `json:"discount_fixed,omitempty" default:"0" nullable:"false"`
	// Customer portal visibility.
	CustomerPortalVisibility *constants.CustomerPortalVisibility `json:"customer_portal_visibility,omitempty" default:"visible" nullable:"false"`
}

var sampleCreateUnitGroupUnitDiscountPct = float64(1)
var sampleCreateUnitGroupUnitDiscountFixed = float64(0)
var sampleCreateUnitGroupUnitVisibility = constants.CustomerPortalVisibilityVisible
var sampleCreateUnitGroupUnitRequest = &CreateUnitGroupUnitRequest{
	UnitID:                   apiresource.SampleUnitID,
	DiscountPercentage:       &sampleCreateUnitGroupUnitDiscountPct,
	DiscountFixed:            &sampleCreateUnitGroupUnitDiscountFixed,
	CustomerPortalVisibility: &sampleCreateUnitGroupUnitVisibility,
}

func (*CreateUnitGroupUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateUnitGroupUnitRequest)
}

type CreateUnitGroupUnitEndpoint struct{}

func (e *CreateUnitGroupUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUnitGroupUnitRequest, *apiresource.UnitGroupUnit] {
	return &apiendpoint.APIEndpoint[*CreateUnitGroupUnitRequest, *apiresource.UnitGroupUnit]{
		Title:             "Create Unit Group Associated Unit",
		Description:       "Creates an associated unit within a unit group.",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/unit-groups/{unit_group_id}/units",
		ContentType:       "application/json",
		Request:           &CreateUnitGroupUnitRequest{},
		Response:          &apiresource.UnitGroupUnit{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateUnitGroupUnitRequest) (*apiresource.UnitGroupUnit, *apierror.APIError) {
			return svc.(UnitGroupSvc).CreateUnitGroupUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnitGroupUnit,
			Fields:     []string{"unit"},
		}),
	}
}
