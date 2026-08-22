package supplierep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/auth-service/pkg/types"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create a supplier.
type CreateSupplierRequest struct {
	// The supplier's name, as shown in the dashboard and on documents.
	Name string `json:"name" validate:"required,max=255"`
	// Human-facing supplier code, such as `SUP-001`.
	//
	// Must be unique per account; creating a supplier with a number already in use returns a conflict error.
	Number string `json:"number" validate:"required,max=255"`
	// Free-form notes about the supplier.
	Note field.Optional[string] `json:"note,omitzero"`
	// Default billing address to create for the supplier.
	//
	// A new address record is created from these values and saved to the supplier; existing addresses cannot be reused here.
	BillToAddress field.Optional[apirequest.AddressInput] `json:"bill_to_address,omitzero"`
	// Default shipping address to create for the supplier.
	//
	// If omitted and `bill_to_address` is provided, the billing address is also used as the default shipping address.
	ShipToAddress field.Optional[apirequest.AddressInput] `json:"ship_to_address,omitzero"`
}

var sampleCreateSupplierNote = "Primary raw materials supplier"
var sampleCreateSupplierStreetLine1 = "456 Industrial Pkwy"
var sampleCreateSupplierLocality = "Chicago"
var sampleCreateSupplierState = "IL"
var sampleCreateSupplierPostalCode = "60601"
var sampleCreateSupplierRequest = &CreateSupplierRequest{
	Name:   apiresource.SampleSupplierName,
	Number: apiresource.SampleSupplierNumber,
	Note:   field.Some(sampleCreateSupplierNote),
	BillToAddress: field.Some(apirequest.AddressInput{
		Name:        apiresource.SampleSupplierName,
		StreetLine1: field.SomePtr(&sampleCreateSupplierStreetLine1),
		Locality:    field.SomePtr(&sampleCreateSupplierLocality),
		State:       field.SomePtr(&sampleCreateSupplierState),
		PostalCode:  field.SomePtr(&sampleCreateSupplierPostalCode),
		Country:     "US",
	}),
}

func (*CreateSupplierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSupplierRequest)
}

// Creates a supplier, optionally with inline bill-to and ship-to addresses.
//
// Returns a conflict error if another supplier in the account already uses the given number.
type CreateSupplierEndpoint struct{}

func (e *CreateSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSupplierRequest, *apiresource.SupplierDetail] {
	return (&apiendpoint.APIEndpoint[*CreateSupplierRequest, *apiresource.SupplierDetail]{
		Title:             "Create Supplier",
		Method:            http.MethodPost,
		Route:             "/v1/operations/suppliers",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
			return svc.(SupplierSvc).CreateSupplier
		},
		LocationFunc: func(resp *apiresource.SupplierDetail) string {
			return "/v1/operations/suppliers/" + resp.ID
		},
	})
}
