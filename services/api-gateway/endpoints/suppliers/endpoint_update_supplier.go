package supplierep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update a supplier.
type UpdateSupplierRequest struct {
	// Supplier ID.
	SupplierID string `path:"id" validate:"required"`
	// The supplier's business name. This is your account's name for the supplier; it does not rename the supplier's own account.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Supplier number, unique within your account.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Whether the supplier can be named on new records: `active` or `archived`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Your team's standing instructions for this supplier's orders. Send `null` to clear them.
	Instructions field.Clearable[string] `json:"instructions,omitzero"`
	// How to reach the supplier's business. Only fields you send are changed.
	Contact field.Optional[SupplierContactInput] `json:"contact,omitzero"`
	// Who pays the carrier for freight from the supplier. Only fields you send are changed.
	CarrierBilling field.Optional[SupplierCarrierBillingInput] `json:"carrier_billing,omitzero"`
	// Values copied onto a new purchase order from the supplier. Only fields you send are changed.
	Defaults field.Optional[UpdateSupplierDefaultsInput] `json:"defaults,omitzero"`
}

var sampleUpdateSupplierRequest = &UpdateSupplierRequest{
	Name:         field.Some("Acme Supplies LLC"),
	Instructions: field.Set("Deliveries to the north dock only."),
}

func (*UpdateSupplierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSupplierRequest)
}

// Updates a supplier.
//
// Only fields you send are changed, including fields inside `contact`, `carrier_billing` and `defaults`. Send `null` to clear a clearable field.
type UpdateSupplierEndpoint struct{}

func (e *UpdateSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSupplierRequest, *apiresource.Supplier] {
	return (&apiendpoint.APIEndpoint[*UpdateSupplierRequest, *apiresource.Supplier]{
		Title:             "Update Supplier",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/suppliers/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSupplier,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
			return svc.(SupplierSvc).UpdateSupplier
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSupplier,
			Fields:     supplierIncludeFields,
		}),
	})
}
