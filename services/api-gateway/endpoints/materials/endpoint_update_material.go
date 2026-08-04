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

// Request to update a material.
type UpdateMaterialRequest struct {
	// ID of the material to update.
	ItemID string `path:"id" validate:"required"`
	// New stock keeping unit code for the material.
	//
	// Must remain unique within the account; a conflict error is returned if another item already uses it.
	SKU field.Optional[string] `json:"sku,omitzero" validate:"omitempty,max=255"`
	// New description for the material.
	Description field.Optional[string] `json:"description,omitzero"`
	// New notes for the material.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// New reorder threshold: when on-hand stock falls to this quantity, the material should be reordered.
	OrderPoint field.Optional[QuantityInputRequest] `json:"order_point,omitzero"`
	// New expected time between placing an order for this material and receiving it.
	LeadTime field.Optional[QuantityInputRequest] `json:"lead_time,omitzero"`
	// New cost per unit.
	//
	// Follows the same unit rule as on create: `numerator_unit_id` must reference a currency unit and `denominator_unit_id` must reference a non-currency unit.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero"`
}

var sampleUpdateMaterialDescription = "Cold-rolled 304 stainless steel sheet, 2.0mm"
var sampleUpdateMaterialNotes = "Reorder point raised after Q2 demand spike."
var sampleUpdateMaterialRequest = &UpdateMaterialRequest{
	SKU:         field.Some("MAT-001-UPDATED"),
	Description: field.Some(sampleUpdateMaterialDescription),
	Notes:       field.Some(sampleUpdateMaterialNotes),
	OrderPoint:  field.Some(QuantityInputRequest{Value: "150.00", UnitID: apiresource.SampleUnitID}),
	LeadTime:    field.Some(QuantityInputRequest{Value: "10.00", UnitID: apiresource.SampleUnitID}),
	UnitCost:    field.Some(apirequest.RateInput{Value: "9.10", NumeratorUnitID: apiresource.SampleUnitID, DenominatorUnitID: apiresource.SampleUnitID}),
}

func (*UpdateMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMaterialRequest)
}

// Partially updates a material.
//
// Fields not provided retain their current values. Only the cost side of pricing can be changed here; the selling price set at creation is not editable through this endpoint. Use the Change Item Category endpoint to move the material to a different category.
type UpdateMaterialEndpoint struct{}

func (e *UpdateMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMaterialRequest, *apiresource.Material] {
	return (&apiendpoint.APIEndpoint[*UpdateMaterialRequest, *apiresource.Material]{
		Title:               "Update Material",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/catalog/materials/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionUpdate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeMaterial,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMaterialRequest) (*apiresource.Material, *apierror.APIError) {
			return svc.(MaterialSvc).UpdateMaterial
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMaterial,
			Fields:     []string{"item", "item.category", "item.category.properties", "item.category.unit_group", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
