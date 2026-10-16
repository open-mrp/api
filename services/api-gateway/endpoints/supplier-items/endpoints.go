package supplieritemep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

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

const supplierItemsRoute = "/v1/operations/suppliers/{supplier_id}/items"

func supplierItemIncludes() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeSupplierItem,
		Fields:     supplierItemIncludeFields,
	})
}

// Request to list a supplier's items.
type ListSupplierItemsRequest struct {
	object.ListRequest
	apirequest.CreatedRange
	// ID of the supplier.
	SupplierID string `path:"supplier_id" validate:"required"`
	// Case-insensitive substring match on `supplier_sku` or `supplier_description`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return supplier items with any of these statuses: `active` or `archived`.
	Statuses []constants.ArchiveStatus `query:"statuses"`
	// Only return supplier items for any of these items.
	ItemIDs []string `query:"item_ids"`
}

// Returns a list of a supplier's items.
//
// Sorted by creation date, newest first.
type ListSupplierItemsEndpoint struct{}

func (e *ListSupplierItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSupplierItemsRequest, *apiresource.List[apiresource.SupplierItem]] {
	return (&apiendpoint.APIEndpoint[*ListSupplierItemsRequest, *apiresource.List[apiresource.SupplierItem]]{
		Title:             "List Supplier Items",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             supplierItemsRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSupplierItem,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSupplierItemsRequest) (*apiresource.List[apiresource.SupplierItem], *apierror.APIError) {
			return svc.(SupplierItemSvc).ListSupplierItems
		},
		IncludeConfig: supplierItemIncludes(),
	})
}

// Request to retrieve a supplier item.
type RetrieveSupplierItemRequest struct {
	// ID of the supplier.
	SupplierID string `path:"supplier_id" validate:"required"`
	// Supplier item ID.
	SupplierItemID string `path:"id" validate:"required"`
}

// Retrieves a supplier item.
type RetrieveSupplierItemEndpoint struct{}

func (e *RetrieveSupplierItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveSupplierItemRequest, *apiresource.SupplierItem] {
	return (&apiendpoint.APIEndpoint[*RetrieveSupplierItemRequest, *apiresource.SupplierItem]{
		Title:             "Retrieve Supplier Item",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             supplierItemsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSupplierItem,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError) {
			return svc.(SupplierItemSvc).GetSupplierItem
		},
		IncludeConfig: supplierItemIncludes(),
	})
}

// Request to create a supplier item.
type CreateSupplierItemRequest struct {
	// ID of the supplier.
	SupplierID string `path:"supplier_id" validate:"required"`
	// ID of the item the supplier provides. It must have a `material` section; a supplier has at most one supplier item per item.
	ItemID string `json:"item_id" validate:"required"`
	// The supplier's own code for this item, as it appears on their documents.
	SupplierSKU string `json:"supplier_sku" validate:"required,max=255"`
	// The supplier's own description of this item, up to 65,535 characters.
	SupplierDescription field.Optional[string] `json:"supplier_description,omitzero" validate:"omitempty,max=65535"`
	// Whether you still buy this item from the supplier: `active` or `archived`. Defaults to `active`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
}

var sampleCreateSupplierItemRequest = &CreateSupplierItemRequest{
	ItemID:              apiresource.SampleItemID,
	SupplierSKU:         "SUP-PART-001",
	SupplierDescription: field.Some("Cold-rolled steel sheet, 16 gauge, 48x96 in."),
}

func (*CreateSupplierItemRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSupplierItemRequest)
}

// Creates a supplier item.
//
// Records that the supplier provides an item, with the supplier's own code for it. Neither the supplier nor the item may be archived.
type CreateSupplierItemEndpoint struct{}

func (e *CreateSupplierItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSupplierItemRequest, *apiresource.SupplierItem] {
	return (&apiendpoint.APIEndpoint[*CreateSupplierItemRequest, *apiresource.SupplierItem]{
		Title:             "Create Supplier Item",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             supplierItemsRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSupplierItem,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSuppliers, Action: types.ActionCreate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError) {
			return svc.(SupplierItemSvc).CreateSupplierItem
		},
		IncludeConfig: supplierItemIncludes(),
	})
}

// Request to update a supplier item.
type UpdateSupplierItemRequest struct {
	// ID of the supplier.
	SupplierID string `path:"supplier_id" validate:"required"`
	// Supplier item ID.
	SupplierItemID string `path:"id" validate:"required"`
	// The supplier's own code for this item.
	SupplierSKU field.Optional[string] `json:"supplier_sku,omitzero" validate:"omitempty,max=255"`
	// The supplier's own description of this item, up to 65,535 characters. Send `null` to clear it.
	SupplierDescription field.Clearable[string] `json:"supplier_description,omitzero" validate:"omitempty,max=65535"`
	// Whether you still buy this item from the supplier: `active` or `archived`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
}

var sampleUpdateSupplierItemRequest = &UpdateSupplierItemRequest{
	SupplierSKU: field.Some("SUP-PART-002"),
}

func (*UpdateSupplierItemRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSupplierItemRequest)
}

// Updates a supplier item.
//
// Only fields you send are changed.
type UpdateSupplierItemEndpoint struct{}

func (e *UpdateSupplierItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSupplierItemRequest, *apiresource.SupplierItem] {
	return (&apiendpoint.APIEndpoint[*UpdateSupplierItemRequest, *apiresource.SupplierItem]{
		Title:             "Update Supplier Item",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             supplierItemsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSupplierItem,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSupplierItemRequest) (*apiresource.SupplierItem, *apierror.APIError) {
			return svc.(SupplierItemSvc).UpdateSupplierItem
		},
		IncludeConfig: supplierItemIncludes(),
	})
}

// Request to delete a supplier item.
type DeleteSupplierItemRequest struct {
	// ID of the supplier.
	SupplierID string `path:"supplier_id" validate:"required"`
	// Supplier item ID.
	SupplierItemID string `path:"id" validate:"required"`
}

// Deletes a supplier item.
//
// Nothing references a supplier item, so it can always be deleted; archive it instead to keep it for reference.
type DeleteSupplierItemEndpoint struct{}

func (e *DeleteSupplierItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSupplierItemRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteSupplierItemRequest, *object.Deleted]{
		Title:             "Delete Supplier Item",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             supplierItemsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSupplierItem,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSupplierItemRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(SupplierItemSvc).DeleteSupplierItem, constants.ObjectTypeSupplierItem, "id")
		},
	})
}
