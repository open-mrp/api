package supplierep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateSupplierRequest is the request to create a supplier.
type CreateSupplierRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Supplier number. Must be unique per account.
	Number string `json:"number" validate:"required,max=255"`
	// Supplier notes.
	Note *string `json:"note"`
	// Bill-to address to create inline.
	BillToAddress *apirequest.AddressInput `json:"bill_to_address"`
	// Ship-to address to create inline.
	ShipToAddress *apirequest.AddressInput `json:"ship_to_address"`
}

var sampleCreateSupplierNote = "Primary raw materials supplier"
var sampleCreateSupplierStreetLine1 = "456 Industrial Pkwy"
var sampleCreateSupplierLocality = "Chicago"
var sampleCreateSupplierState = "IL"
var sampleCreateSupplierPostalCode = "60601"
var sampleCreateSupplierRequest = &CreateSupplierRequest{
	Name:   apiresource.SampleSupplierName,
	Number: apiresource.SampleSupplierNumber,
	Note:   &sampleCreateSupplierNote,
	BillToAddress: &apirequest.AddressInput{
		Name:        apiresource.SampleSupplierName,
		StreetLine1: &sampleCreateSupplierStreetLine1,
		Locality:    &sampleCreateSupplierLocality,
		State:       &sampleCreateSupplierState,
		PostalCode:  &sampleCreateSupplierPostalCode,
		Country:     "US",
	},
}

func (*CreateSupplierRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSupplierRequest)
}

type CreateSupplierEndpoint struct{}

func (e *CreateSupplierEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSupplierRequest, *apiresource.SupplierDetail] {
	return &apiendpoint.APIEndpoint[*CreateSupplierRequest, *apiresource.SupplierDetail]{
		Title:             "Create Supplier",
		Description:       "Creates a supplier, optionally with inline bill-to and ship-to addresses.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/suppliers",
		ContentType:       "application/json",
		Request:           &CreateSupplierRequest{},
		Response:          &apiresource.SupplierDetail{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
			return svc.(SupplierSvc).CreateSupplier
		},
		LocationFunc: func(resp *apiresource.SupplierDetail) string {
			return "/v1/operations/suppliers/" + resp.ID
		},
	}
}
