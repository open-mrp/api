package invoiceep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get an invoice.
type RetrieveInvoiceRequest struct {
	// Invoice ID.
	InvoiceID string `path:"id" validate:"required"`
}

// Returns an invoice by ID.
type RetrieveInvoiceEndpoint struct{}

func (e *RetrieveInvoiceEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveInvoiceRequest, *apiresource.Invoice] {
	return (&apiendpoint.APIEndpoint[*RetrieveInvoiceRequest, *apiresource.Invoice]{
		Title:             "Retrieve Invoice",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/finance/invoices/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeInvoice,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInvoices, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveInvoiceRequest) (*apiresource.Invoice, *apierror.APIError) {
			return svc.(InvoiceSvc).GetInvoice
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInvoice,
			Fields: []string{
				"customer",
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
