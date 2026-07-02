package materialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// A quantity, given as a decimal value and the unit it is measured in.
type QuantityInputRequest struct {
	// Decimal value of the quantity.
	Value string `json:"value" validate:"required"`
	// ID of the unit the value is expressed in.
	UnitID string `json:"unit_id" validate:"required"`
}

// Request to create a material.
type CreateMaterialRequest struct {
	// Stock keeping unit code for the material.
	//
	// Must be unique within the account; creating a material with a SKU already used by another item fails with a conflict error.
	SKU string `json:"sku" validate:"required,max=255"`
	// Free-form description of the material.
	Description field.Optional[string] `json:"description,omitzero"`
	// Free-form notes about the material.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// ID of the item category to place the material in.
	//
	// The category's unit group determines the base unit used for the material's rates (`unit_value`, `unit_cost`, `burn_rate`).
	CategoryID string `json:"category_id" validate:"required"`
	// Reorder threshold: when on-hand stock falls to this quantity, the material should be reordered.
	//
	// When omitted, the order point is initialized to a zero quantity in the category's base unit.
	OrderPoint field.Optional[QuantityInputRequest] `json:"order_point,omitzero"`
	// Expected time between placing an order for this material and receiving it, expressed as a quantity in a time unit (e.g. days).
	//
	// When omitted, the lead time is initialized to a zero quantity in the category's base unit.
	LeadTime field.Optional[QuantityInputRequest] `json:"lead_time,omitzero"`
	// Initial selling price per unit.
	//
	// `numerator_unit_id` must reference a currency unit and `denominator_unit_id` must reference a non-currency unit (e.g. `$5` per `ea`). When omitted, the price is initialized to a zero rate in the category's base unit.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Initial cost per unit.
	//
	// Follows the same unit rule as `unit_price`: currency numerator, non-currency denominator. When omitted, the cost is initialized to a zero rate in the category's base unit.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero"`
	// IDs of existing attributes to link to the material at creation time.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
}

var sampleCreateMaterialRequest = &CreateMaterialRequest{
	SKU:        "MAT-001",
	CategoryID: apiresource.SampleItemCategoryID,
}

func (*CreateMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMaterialRequest)
}

// Creates a material with the specified SKU and category.
//
// Inventory tracking for the new material starts at a zero on-hand quantity in the category's base unit.
type CreateMaterialEndpoint struct{}

func (e *CreateMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMaterialRequest, *apiresource.Material] {
	return (&apiendpoint.APIEndpoint[*CreateMaterialRequest, *apiresource.Material]{
		Title:               "Create Material",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/catalog/materials",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionCreate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeMaterial,
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
