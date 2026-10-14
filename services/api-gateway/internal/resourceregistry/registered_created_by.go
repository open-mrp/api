package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

// ObjectTypeCreatedBy is a derived sub-resource (not a top-level endpoint): it only hosts the loader that resolves a resource's creator from its create audit event. The loader is keyed by the PARENT resource's ID (e.g. the sales order id), not by a created_by id. It targets sales-order create events, then purchase-order ones for ids with no sales-order event, so a resource that borrows its creator from an order — a pick, say — keys the include by that order's id rather than its own.
func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeCreatedBy,
		Load:       resourceloaders.LoadCreatedBySalesOrders,
	})
}
