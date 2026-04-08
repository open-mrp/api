package quantityep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateQuantityRequest is the request to partially update a quantity.
type UpdateQuantityRequest struct {
	// The ID of the quantity to update.
	QuantityID string `path:"id" validate:"required"`
	// The new decimal value of the quantity.
	Value *string `json:"value,omitempty"`
	// The new unit ID for this quantity.
	UnitID *string `json:"unit_id,omitempty" validate:"omitempty,max=191"`
	// The ID of the parent resource that owns this quantity.
	ObjectID *string `json:"object_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
	// The type of the parent resource (e.g. "item", "production_step").
	ObjectType *string `json:"object_type,omitempty" nullable:"false" validate:"omitempty,max=255"`
}

var sampleUpdateQuantityValue = "50.000000000000000000000000000000"
var sampleUpdateQuantityUnitID = apiresource.SampleUnitID
var sampleUpdateQuantityRequest = &UpdateQuantityRequest{
	Value:  &sampleUpdateQuantityValue,
	UnitID: &sampleUpdateQuantityUnitID,
}

func (*UpdateQuantityRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateQuantityRequest)
}

type UpdateQuantityEndpoint struct{}

func (e *UpdateQuantityEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateQuantityRequest, *apiresource.Quantity] {
	return &apiendpoint.APIEndpoint[*UpdateQuantityRequest, *apiresource.Quantity]{
		Title:             "Update Quantity",
		Description:       "Partially updates a quantity.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/quantities/{id}",
		ContentType:       "application/json",
		Request:           &UpdateQuantityRequest{},
		Response:          &apiresource.Quantity{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateQuantityRequest) (*apiresource.Quantity, *apierror.APIError) {
			return svc.(QuantitySvc).UpdateQuantity
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeQuantity,
			Fields:     []string{"unit"},
		}),
	}
}
