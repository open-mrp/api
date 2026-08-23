package quantityep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
)

// Request to partially update a quantity.
type UpdateQuantityRequest struct {
	// Quantity ID.
	QuantityID string `path:"id" validate:"required"`
	// New decimal value for the quantity, as a string to preserve precision.
	Value field.Optional[string] `json:"value,omitzero"`
	// ID of the new unit of measure for the quantity.
	//
	// The stored value is kept as-is and is not converted into the new unit, so send `value` alongside this when the amount should change too.
	UnitID field.Optional[string] `json:"unit_id,omitzero" validate:"omitempty"`
	// ID of the resource that owns this quantity.
	//
	// Used together with `object_type` to verify the owning resource exists; it does not reassign the quantity.
	ObjectID field.Optional[string] `json:"object_id,omitzero" validate:"omitempty"`
	// Type of the resource that owns this quantity.
	//
	// Determines the permission required for the update.
	ObjectType field.Optional[constants.MeasureOwnerType] `json:"object_type,omitzero" validate:"omitempty"`
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

// Updates the value or unit of a quantity in place.
//
// A quantity belongs to the resource that reports it — a material's order point, the amount a production step consumes, and so on — so this changes that resource's stored measure directly.
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
