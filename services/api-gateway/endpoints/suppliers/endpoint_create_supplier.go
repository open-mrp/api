package supplierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// CreateSupplierRequest is the request to create a supplier.
type CreateSupplierRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Supplier number. Must be unique per account.
	Number string `json:"number" validate:"required,max=255"`
	// Supplier notes.
	Note field.Optional[string] `json:"note,omitzero"`
	// Bill-to address to create inline.
	BillToAddress field.Optional[apirequest.AddressInput] `json:"bill_to_address,omitzero"`
	// Ship-to address to create inline.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
			return svc.(SupplierSvc).CreateSupplier
		},
		LocationFunc: func(resp *apiresource.SupplierDetail) string {
			return "/v1/operations/suppliers/" + resp.ID
		},
	})
}
