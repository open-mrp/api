package quantityep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"

	"github.com/augno/api/services/auth-service/pkg/types"
)

// Request to partially update a quantity.
type UpdateQuantityRequest struct {
	// Quantity ID.
	QuantityID string `path:"id" validate:"required"`
	// New decimal value for the quantity, as a string to preserve precision.
	Value field.Optional[string] `json:"value,omitzero"`
	// ID of the new unit of measure for the quantity.
	UnitID field.Optional[string] `json:"unit_id,omitzero" validate:"omitempty"`
	// ID of the resource that owns this quantity.
	//
	// Used together with `object_type` to verify the owning resource exists; it does not reassign the quantity.
	ObjectID field.Optional[string] `json:"object_id,omitzero" validate:"omitempty"`
	// Type of the resource that owns this quantity.
	//
	// Determines the permission required for the update. Must be `item` or `production_step`.
	ObjectType field.Optional[string] `json:"object_type,omitzero" validate:"omitempty,max=255"`
}

var sampleUpdateQuantityValue = "50.000000000000000000000000000000"
var sampleUpdateQuantityUnitID = apiresource.SampleUnitID
var sampleUpdateQuantityRequest = &UpdateQuantityRequest{
	Value:  field.Some(sampleUpdateQuantityValue),
	UnitID: field.Some(sampleUpdateQuantityUnitID),
}

func (*UpdateQuantityRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateQuantityRequest)
}

// Partially updates a quantity.
type UpdateQuantityEndpoint struct{}

func (e *UpdateQuantityEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateQuantityRequest, *apiresource.Quantity] {
	return (&apiendpoint.APIEndpoint[*UpdateQuantityRequest, *apiresource.Quantity]{
		Title:             "Update Quantity",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/quantities/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainItems, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainProductionSteps, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeQuantity,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateQuantityRequest) (*apiresource.Quantity, *apierror.APIError) {
			return svc.(QuantitySvc).UpdateQuantity
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeQuantity,
			Fields:     []string{"unit"},
		}),
	})
}
