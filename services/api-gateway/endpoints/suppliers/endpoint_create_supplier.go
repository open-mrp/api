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

// CreateSupplierRequest is the request to create a new supplier.
type CreateSupplierRequest struct {
	// The display name of the supplier.
	Name string `json:"name" validate:"required"`
	// The supplier number (must be unique per account).
	Number string `json:"number" validate:"required"`
	// Notes about the supplier.
	Note *string `json:"note"`
	// An optional bill-to address to create inline.
	BillToAddress *apirequest.AddressInput `json:"bill_to_address"`
	// An optional ship-to address to create inline.
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
		Description:       "Creates a new supplier, optionally with inline bill-to and ship-to addresses.",
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
	}
}
