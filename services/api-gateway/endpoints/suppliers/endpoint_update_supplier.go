package supplierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// UpdateSupplierRequest is the request to update a supplier.
type UpdateSupplierRequest struct {
	// Supplier ID.
	SupplierID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Supplier number.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Note value. Set update_note to true to apply.
	Note field.Optional[string] `json:"note,omitzero"`
	// Whether to update the note field. Allows clearing to null.
	UpdateNote bool `json:"update_note"`
	// Bill-to address ID.
	BillToAddressID field.Optional[string] `json:"bill_to_address_id,omitzero" validate:"omitempty"`
	// Ship-to address ID.
	ShipToAddressID field.Optional[string] `json:"ship_to_address_id,omitzero" validate:"omitempty"`
}

var sampleUpdateSupplierName = "Acme Supplies LLC"
var sampleUpdateSupplierNote = "Updated contact info"
var sampleUpdateSupplierRequest = &UpdateSupplierRequest{
	Name:       field.Some(sampleUpdateSupplierName),
	Note:       field.Some(sampleUpdateSupplierNote),
	UpdateNote: true,
}

func (*UpdateSupplierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSupplierRequest)
}

// Partially updates a supplier. Set update_note to true to update the note field, including clearing it.
type UpdateSupplierEndpoint struct{}

func (e *UpdateSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSupplierRequest, *apiresource.SupplierDetail] {
	return (&apiendpoint.APIEndpoint[*UpdateSupplierRequest, *apiresource.SupplierDetail]{
		Title:             "Update Supplier",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/suppliers/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
			return svc.(SupplierSvc).UpdateSupplier
		},
	})
}
