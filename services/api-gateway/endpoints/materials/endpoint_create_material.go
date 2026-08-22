package materialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// A quantity, given as a decimal value and the unit it is measured in.
type QuantityInputRequest struct {
	// Decimal value of the quantity.
	Value string `json:"value" validate:"required" format:"decimal"`
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
	// When omitted, the material is created without a reorder threshold.
	OrderPoint field.Optional[QuantityInputRequest] `json:"order_point,omitzero"`
	// Expected time between placing an order for this material and receiving it, expressed as a quantity in a time unit (e.g. days).
	//
	// When omitted, the material is created without a lead time.
	LeadTime field.Optional[QuantityInputRequest] `json:"lead_time,omitzero"`
	// Initial selling price per unit.
	//
	// `numerator_unit_id` must reference a currency unit and `denominator_unit_id` must reference a non-currency unit (e.g. `$5` per `ea`). When omitted, the price is initialized to a zero rate in the category's base unit. It becomes the `unit_value` rate on the material's item; the material update endpoint cannot change it afterwards.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Initial cost per unit.
	//
	// Follows the same unit rule as `unit_price`: currency numerator, non-currency denominator. When omitted, the cost is initialized to a zero rate in the category's base unit.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero"`
	// IDs of existing attributes to link to the material at creation time.
	//
	// Each attribute's property must be one the material's category carries; an attribute from any other property fails the whole request.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
}

var sampleCreateMaterialDescription = "Cold-rolled 304 stainless steel sheet, 1.5mm"
var sampleCreateMaterialNotes = "Store flat in a dry area to avoid surface oxidation."
var sampleCreateMaterialRequest = &CreateMaterialRequest{
	SKU:          "MAT-001",
	Description:  field.Some(sampleCreateMaterialDescription),
	Notes:        field.Some(sampleCreateMaterialNotes),
	CategoryID:   apiresource.SampleItemCategoryID,
	OrderPoint:   field.Some(QuantityInputRequest{Value: "100.00", UnitID: apiresource.SampleUnitID}),
	LeadTime:     field.Some(QuantityInputRequest{Value: "7.00", UnitID: apiresource.SampleUnitID}),
	UnitPrice:    field.Some(apirequest.RateInput{Value: "12.50", NumeratorUnitID: apiresource.SampleUnitID, DenominatorUnitID: apiresource.SampleUnitID}),
	UnitCost:     field.Some(apirequest.RateInput{Value: "8.25", NumeratorUnitID: apiresource.SampleUnitID, DenominatorUnitID: apiresource.SampleUnitID}),
	AttributeIDs: []string{apiresource.SampleAttributeID},
}

func (*CreateMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMaterialRequest)
}

// Creates a material together with the catalog item that carries its SKU, description, category, pricing, and attributes.
//
// Inventory tracking for the new material starts at a zero on-hand quantity in the category's base unit. The item's consumption rate (`burn_rate`) also starts at zero and cannot be supplied here — it is derived from recorded consumption as production happens.
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
