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
	"github.com/augno/api/shared/field"
)

// QuantityInputRequest is a quantity value and unit.
type QuantityInputRequest struct {
	// Quantity value.
	Value string `json:"value" validate:"required"`
	// Unit ID.
	UnitID string `json:"unit_id" validate:"required"`
}

// Request to create a material.
type CreateMaterialRequest struct {
	// SKU code.
	SKU string `json:"sku" validate:"required,max=255"`
	// Description.
	Description field.Optional[string] `json:"description,omitzero"`
	// Notes.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Category ID.
	CategoryID string `json:"category_id" validate:"required"`
	// Order point quantity.
	OrderPoint field.Optional[QuantityInputRequest] `json:"order_point,omitzero"`
	// Lead time quantity.
	LeadTime field.Optional[QuantityInputRequest] `json:"lead_time,omitzero"`
	// Initial unit price. When set, numerator must be a currency unit and
	// denominator must not be.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Initial unit cost. Same currency rule as unit_price.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero"`
	// Attribute IDs to connect to the material at creation time.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
}

var sampleCreateMaterialRequest = &CreateMaterialRequest{
	SKU:        "MAT-001",
	CategoryID: apiresource.SampleItemCategoryID,
}

func (*CreateMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMaterialRequest)
}

// Creates a material.
type CreateMaterialEndpoint struct{}

func (e *CreateMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMaterialRequest, *apiresource.Material] {
	return (&apiendpoint.APIEndpoint[*CreateMaterialRequest, *apiresource.Material]{
		Title:             "Create Material",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/materials",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeMaterial,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMaterialRequest) (*apiresource.Material, *apierror.APIError) {
			return svc.(MaterialSvc).CreateMaterial
		},
		LocationFunc: func(resp *apiresource.Material) string {
			return "/v1/catalog/materials/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMaterial,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
