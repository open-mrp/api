package partep

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

// Request to partially update a part.
type UpdatePartRequest struct {
	// ID of the part to update.
	ItemID string `path:"id" validate:"required"`
	// New stock keeping unit code for the part.
	//
	// Must remain unique within the account; a conflict error is returned if another item already uses it.
	SKU field.Optional[string] `json:"sku,omitzero" validate:"omitempty,max=255"`
	// New description for the part.
	//
	// Set to a string to replace the current description, or `null` to clear it.
	Description field.Clearable[string] `json:"description,omitzero"`
	// New notes for the part.
	//
	// Set to a string to replace the current notes, or `null` to clear them.
	Notes field.Clearable[string] `json:"notes,omitzero"`
}

var sampleUpdatePartSKU = apiresource.SamplePartSKU
var sampleUpdatePartDescription = "Deep groove ball bearing, 20x47x14mm"
var sampleUpdatePartNotes = "Superseded by low-friction variant; keep for legacy assemblies."
var sampleUpdatePartRequest = &UpdatePartRequest{
	SKU:         field.SomePtr(&sampleUpdatePartSKU),
	Description: field.Set(sampleUpdatePartDescription),
	Notes:       field.Set(sampleUpdatePartNotes),
}

func (*UpdatePartRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePartRequest)
}

// Partially updates a part.
//
// Fields not provided retain their current values.
type UpdatePartEndpoint struct{}

func (e *UpdatePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePartRequest, *apiresource.Part] {
	return (&apiendpoint.APIEndpoint[*UpdatePartRequest, *apiresource.Part]{
		Title:               "Update Part",
		Method:              http.MethodPatch,
		Route:               "/v1/catalog/parts/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainParts, Action: types.ActionUpdate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).UpdatePart
		},
		ObjectType: constants.ObjectTypePart,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
