package materialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a material.
type UpdateMaterialRequest struct {
	// Material ID.
	ItemID string `path:"id" validate:"required"`
	// SKU code.
	SKU *string `json:"sku,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Description.
	Description *string `json:"description,omitempty" nullable:"false"`
	// Notes.
	Notes *string `json:"notes,omitempty" nullable:"false"`
	// Order point quantity.
	OrderPoint *QuantityInputRequest `json:"order_point,omitempty" nullable:"false"`
	// Lead time quantity.
	LeadTime *QuantityInputRequest `json:"lead_time,omitempty" nullable:"false"`
	// Updated unit cost. Same currency rule as on create.
	UnitCost *apirequest.RateInput `json:"unit_cost,omitempty"`
}

var sampleUpdateMaterialSKU = "MAT-001-UPDATED"
var sampleUpdateMaterialRequest = &UpdateMaterialRequest{
	SKU: &sampleUpdateMaterialSKU,
}

func (*UpdateMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMaterialRequest)
}

// Partially updates a material.
type UpdateMaterialEndpoint struct{}

func (e *UpdateMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMaterialRequest, *apiresource.Material] {
	return (&apiendpoint.APIEndpoint[*UpdateMaterialRequest, *apiresource.Material]{
		Title:             "Update Material",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/catalog/materials/{id}",
		Request:           &UpdateMaterialRequest{},
		Response:          &apiresource.Material{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMaterialRequest) (*apiresource.Material, *apierror.APIError) {
			return svc.(MaterialSvc).UpdateMaterial
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMaterial,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	}).WithDocSource(e)
}
