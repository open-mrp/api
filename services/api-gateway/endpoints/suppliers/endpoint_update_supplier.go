package supplierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateSupplierRequest is the request to update a supplier.
type UpdateSupplierRequest struct {
	// The ID of the supplier to update.
	SupplierID string `path:"id" validate:"required"`
	// The new display name.
	Name *string `json:"name" validate:"omitempty,max=255"`
	// The new supplier number.
	Number *string `json:"number" validate:"omitempty,max=255"`
	// The new note. Set update_note to true to apply.
	Note *string `json:"note"`
	// Whether to update the note field (allows setting to null).
	UpdateNote bool `json:"update_note"`
	// The ID of the bill-to address to set.
	BillToAddressID *string `json:"bill_to_address_id" nullable:"true" validate:"omitempty,max=191"`
	// The ID of the ship-to address to set.
	ShipToAddressID *string `json:"ship_to_address_id" nullable:"true" validate:"omitempty,max=191"`
}

var sampleUpdateSupplierName = "Acme Supplies LLC"
var sampleUpdateSupplierNote = "Updated contact info"
var sampleUpdateSupplierRequest = &UpdateSupplierRequest{
	Name:       &sampleUpdateSupplierName,
	Note:       &sampleUpdateSupplierNote,
	UpdateNote: true,
}

func (*UpdateSupplierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSupplierRequest)
}

type UpdateSupplierEndpoint struct{}

func (e *UpdateSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSupplierRequest, *apiresource.SupplierDetail] {
	return &apiendpoint.APIEndpoint[*UpdateSupplierRequest, *apiresource.SupplierDetail]{
		Title:             "Update Supplier",
		Description:       "Partially updates a supplier. Set update_note to true to update the note field, including clearing it.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/suppliers/{id}",
		ContentType:       "application/json",
		Request:           &UpdateSupplierRequest{},
		Response:          &apiresource.SupplierDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
			return svc.(SupplierSvc).UpdateSupplier
		},
	}
}
