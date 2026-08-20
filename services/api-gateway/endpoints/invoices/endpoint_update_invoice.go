package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update an invoice.
type UpdateInvoiceRequest struct {
	// Invoice ID.
	InvoiceID string `path:"id" validate:"required"`
	// Free-text note attached to the invoice; send `null` to clear it.
	Note field.Clearable[string] `json:"note,omitzero"`
	// Records whether the invoice has been sent to the customer.
	//
	// Emailing the invoice through Email Record sets this on its own, so it only needs to be set here when the invoice was delivered outside the platform.
	HasBeenSent field.Optional[bool] `json:"has_been_sent,omitzero"`
	// Records whether the invoice has been transmitted to the customer via EDI.
	//
	// A tracking flag only; setting it does not transmit anything.
	IsEdiSent field.Optional[bool] `json:"is_edi_sent,omitzero"`
	// Whether the invoice has been paid in full.
	//
	// Setting this to `true` marks the invoice as paid regardless of the payments recorded against it, which updates the invoice's `payment_status` and drops it from receivables listings. Recording a settlement against the invoice later recalculates the flag from its allocations and can overwrite the value set here.
	IsPaidInFull field.Optional[bool] `json:"is_paid_in_full,omitzero"`
}

var sampleUpdateInvoiceNote = "Payment received via wire transfer"
var sampleUpdateInvoiceHasBeenSent = true
var sampleUpdateInvoiceRequest = &UpdateInvoiceRequest{
	Note:        field.Set(sampleUpdateInvoiceNote),
	HasBeenSent: field.Some(sampleUpdateInvoiceHasBeenSent),
}

func (*UpdateInvoiceRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateInvoiceRequest)
}

// Updates an invoice's note and its sent and paid tracking flags.
//
// Only the fields supplied in the request are changed. The invoice's lines, its customer, and the amounts it bills follow the sales order behind the invoice and cannot be changed here.
type UpdateInvoiceEndpoint struct{}

func (e *UpdateInvoiceEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateInvoiceRequest, *apiresource.Invoice] {
	return (&apiendpoint.APIEndpoint[*UpdateInvoiceRequest, *apiresource.Invoice]{
		Title:             "Update Invoice",
		Method:            http.MethodPatch,
		Route:             "/v1/finance/invoices/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeInvoice,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInvoices, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateInvoiceRequest) (*apiresource.Invoice, *apierror.APIError) {
			return svc.(InvoiceSvc).UpdateInvoice
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInvoice,
			Fields: []string{"customer",
				"order",
				"shipment",
				"related.sales_order",
				"related.shipment",
				"billing_address",
				"payment_term",
				"lines",
				"lines.order_line",
				"lines.order_line.product",
				"allocations",
			},
		}),
	})
}
