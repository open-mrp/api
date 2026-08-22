package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

// ObjectTypeCreatedBy is a derived sub-resource (not a top-level endpoint): it only hosts the loader that resolves a resource's creator from its create audit event. The loader is keyed by the PARENT resource's ID (e.g. the sales order id), not by a created_by id. Today only sales orders expose created_by, so the loader targets sales-order create events.
func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeCreatedBy,
		Load:       resourceloaders.LoadCreatedBySalesOrders,
	})
}
