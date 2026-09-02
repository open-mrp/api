package supplierep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/auth-service/pkg/types"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a supplier.
type UpdateSupplierRequest struct {
	// Supplier ID.
	SupplierID string `path:"id" validate:"required"`
	// The supplier's name, as shown in the dashboard and on documents.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Human-facing supplier code, such as `SUP-001`.
	//
	// Must be unique per account; updating to a number already used by another supplier returns a conflict error.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// New value for the supplier's note.
	//
	// Ignored unless `update_note` is `true`.
	Note field.Optional[string] `json:"note,omitzero"`
	// Whether to apply the `note` field.
	//
	// When `true`, the note is set to the provided `note` value, or cleared if `note` is omitted. When `false`, the note is left unchanged.
	UpdateNote bool `json:"update_note"`
	// ID of an existing address to set as the supplier's default billing address.
	BillToAddressID field.Optional[string] `json:"bill_to_address_id,omitzero" validate:"omitempty"`
	// ID of an existing address to set as the supplier's default shipping address.
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

// Partially updates a supplier.
//
// Only provided fields are changed. To update or clear the note, set `update_note` to `true`.
type UpdateSupplierEndpoint struct{}

func (e *UpdateSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSupplierRequest, *apiresource.Supplier] {
	return (&apiendpoint.APIEndpoint[*UpdateSupplierRequest, *apiresource.Supplier]{
		Title:             "Update Supplier",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/suppliers/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
			return svc.(SupplierSvc).UpdateSupplier
		},
	})
}
