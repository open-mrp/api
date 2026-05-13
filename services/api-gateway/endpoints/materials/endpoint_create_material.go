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
	Description *string `json:"description,omitempty"`
	// Notes.
	Notes *string `json:"notes,omitempty"`
	// Category ID.
	CategoryID string `json:"category_id" validate:"required"`
	// Order point quantity.
	OrderPoint *QuantityInputRequest `json:"order_point,omitempty"`
	// Lead time quantity.
	LeadTime *QuantityInputRequest `json:"lead_time,omitempty"`
	// Initial unit price. When set, numerator must be a currency unit and
	// denominator must not be.
	UnitPrice *apirequest.RateInput `json:"unit_price,omitempty"`
	// Initial unit cost. Same currency rule as unit_price.
	UnitCost *apirequest.RateInput `json:"unit_cost,omitempty"`
	// Initial burn rate (waste / scrap). No currency requirement.
	BurnRate *apirequest.RateInput `json:"burn_rate,omitempty"`
	// Attribute IDs to connect to the material at creation time.
	AttributeIDs []string `json:"attribute_ids,omitempty"`
}

var sampleCreateMaterialRequest = &CreateMaterialRequest{
	SKU:        "MAT-001",
	CategoryID: apiresource.SampleItemCategoryID,
}

func (*CreateMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMaterialRequest)
}

type CreateMaterialEndpoint struct{}

func (e *CreateMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMaterialRequest, *apiresource.Material] {
	return &apiendpoint.APIEndpoint[*CreateMaterialRequest, *apiresource.Material]{
		Title:             "Create Material",
		Description:       "Creates a material.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/materials",
		Request:           &CreateMaterialRequest{},
		Response:          &apiresource.Material{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
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
	}
}
