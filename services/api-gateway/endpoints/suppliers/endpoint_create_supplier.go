package supplierep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to create a supplier.
type CreateSupplierRequest struct {
	// The supplier's business name, as shown throughout the app and on documents.
	Name string `json:"name" validate:"required,max=255"`
	// Supplier number, unique within your account, such as `SUP-001`.
	Number string `json:"number" validate:"required,max=255"`
	// Whether the supplier can be named on new records: `active` or `archived`. Defaults to `active`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Your team's standing instructions for this supplier's orders.
	Instructions field.Optional[string] `json:"instructions,omitzero"`
	// How to reach the supplier's business.
	Contact field.Optional[SupplierContactInput] `json:"contact,omitzero"`
	// Who pays the carrier for freight from the supplier.
	CarrierBilling field.Optional[SupplierCarrierBillingInput] `json:"carrier_billing,omitzero"`
	// Values copied onto a new purchase order from the supplier when the order doesn't set them.
	Defaults field.Optional[CreateSupplierDefaultsInput] `json:"defaults,omitzero"`
	// The supplier's billing address. It becomes `defaults.bill_to_address`.
	BillToAddress field.Optional[apirequest.AddressInput] `json:"bill_to_address,omitzero"`
	// The address the supplier ships from. It becomes `defaults.ship_to_address`; omitted, or identical to the billing address, it is the billing address.
	ShipToAddress field.Optional[apirequest.AddressInput] `json:"ship_to_address,omitzero"`
}

var sampleCreateSupplierStreetLine1 = "456 Industrial Pkwy"
var sampleCreateSupplierLocality = "Chicago"
var sampleCreateSupplierState = "IL"
var sampleCreateSupplierPostalCode = "60601"
var sampleCreateSupplierRequest = &CreateSupplierRequest{
	Name:         apiresource.SampleSupplierName,
	Number:       apiresource.SampleSupplierNumber,
	Instructions: field.Some("Confirm every delivery date by email."),
	Contact: field.Some(SupplierContactInput{
		Email: field.Set("orders@acmesupplies.com"),
		Phone: field.Set("555-987-6543"),
	}),
	Defaults: field.Some(CreateSupplierDefaultsInput{
		PaymentTermID:  field.Some(apiresource.SamplePaymentTermID),
		ShippingTermID: field.Some(apiresource.SampleShippingTermID),
	}),
	BillToAddress: field.Some(apirequest.AddressInput{
		Name:       apiresource.SampleSupplierName,
		Line1:      field.SomePtr(&sampleCreateSupplierStreetLine1),
		City:       field.SomePtr(&sampleCreateSupplierLocality),
		State:      field.SomePtr(&sampleCreateSupplierState),
		PostalCode: field.SomePtr(&sampleCreateSupplierPostalCode),
		Country:    "US",
	}),
}

func (*CreateSupplierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSupplierRequest)
}

// Creates a supplier.
//
// Give its billing and ship-from addresses inline; they become the supplier's default addresses. `number` must be unique within your account.
type CreateSupplierEndpoint struct{}

func (e *CreateSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSupplierRequest, *apiresource.Supplier] {
	return (&apiendpoint.APIEndpoint[*CreateSupplierRequest, *apiresource.Supplier]{
		Title:             "Create Supplier",
		Method:            http.MethodPost,
		Route:             "/v1/operations/suppliers",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSupplier,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionCreate},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
			return svc.(SupplierSvc).CreateSupplier
		},
		LocationFunc: func(resp *apiresource.Supplier) string {
			return "/v1/operations/suppliers/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSupplier,
			Fields:     supplierIncludeFields,
		}),
	})
}
