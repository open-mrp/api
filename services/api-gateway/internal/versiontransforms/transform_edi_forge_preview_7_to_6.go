package versiontransforms

import (
	"net/http"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&ediForgePreview7To6{})
}

// ediForgePreview7To6 bridges 1.0.forge-preview.7 and 1.0.forge-preview.6 for the EDI fields OpenMRP no
// longer stores.
//
// preview.7 removes a customer's `edi_status` and an invoice's `is_edi_sent` and `customer_is_edi_enabled`:
// OpenMRP no longer exchanges EDI documents itself, and the columns behind them are dropped. With no data
// left, older versions lose the response fields too rather than being sent a value nobody recorded. A
// preview.6 request that still sends `edi_status` or `is_edi_sent` has it dropped instead of rejected as an
// unknown field.
type ediForgePreview7To6 struct{}

const (
	customerCreateRoute = "/v1/sales/customers"
	customerUpdateRoute = "/v1/sales/customers/{id}"
	invoiceUpdateRoute  = "/v1/finance/invoices/{id}"
)

func (t *ediForgePreview7To6) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview7
}

func (t *ediForgePreview7To6) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview6
}

func (t *ediForgePreview7To6) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{constants.ObjectTypeCustomer, constants.ObjectTypeInvoice}
}

func (t *ediForgePreview7To6) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

func (t *ediForgePreview7To6) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	// Superseded by TransformRequestBody, which knows which endpoint the body is for.
	return data
}

func (t *ediForgePreview7To6) TransformRequestBody(_ constants.ObjectType, method, route string, data map[string]any) map[string]any {
	switch {
	case method == http.MethodPost && route == customerCreateRoute,
		method == http.MethodPatch && route == customerUpdateRoute:
		delete(data, "edi_status")
	case method == http.MethodPatch && route == invoiceUpdateRoute:
		delete(data, "is_edi_sent")
	}
	return data
}
