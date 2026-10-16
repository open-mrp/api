package inventorymovementep

import (
	"context"
	"net/http"
	"time"

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
	httptransport "github.com/open-mrp/apikit/transport"
)

const movementsRoute = "/v1/operations/inventory-movements"

func movementIncludes() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeInventoryMovement,
		Fields:     []string{"item", "location", "lot", "related"},
	})
}

var readMovements = authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainInventoryLogs, Action: types.ActionRead}}}

// Request to list inventory movements.
type ListInventoryMovementsRequest struct {
	object.ListRequest
	// Only return movements of any of these items.
	ItemIDs []string `query:"item_ids"`
	// Only return movements of any of these types.
	Types []constants.InventoryMovementType `query:"types"`
	// Only return movements caused by any of these users, API keys, agents or devices.
	ActorIDs []string `query:"actor_ids"`
	// Only return movements caused by any of these kinds of actor: `user`, `api_key`, `agent`, `device` or `system`.
	ActorTypes []constants.ActorType `query:"actor_types"`
	// Only return movements at any of these locations.
	LocationIDs []string `query:"location_ids"`
	// Only return movements that occurred at or after this instant.
	//
	// Defaults to 90 days before `occurred_before` (or now) unless `item_ids` is given, in which case the items' whole history is returned.
	OccurredAfter *time.Time `query:"occurred_after"`
	// Only return movements that occurred at or before this instant.
	OccurredBefore *time.Time `query:"occurred_before"`
}

// Returns a list of inventory movements.
//
// Sorted by when they occurred, newest first. Without `occurred_after` or `item_ids`, only the last 90 days are listed.
type ListInventoryMovementsEndpoint struct{}

func (e *ListInventoryMovementsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListInventoryMovementsRequest, *apiresource.List[apiresource.InventoryMovement]] {
	return (&apiendpoint.APIEndpoint[*ListInventoryMovementsRequest, *apiresource.List[apiresource.InventoryMovement]]{
		Title:             "List Inventory Movements",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             movementsRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeInventoryMovement,
		Auth:              readMovements,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListInventoryMovementsRequest) (*apiresource.List[apiresource.InventoryMovement], *apierror.APIError) {
			return svc.(InventoryMovementSvc).ListInventoryMovements
		},
		IncludeConfig: movementIncludes(),
	})
}

// Request to retrieve an inventory movement.
type RetrieveInventoryMovementRequest struct {
	// Inventory movement ID.
	InventoryMovementID string `path:"id" validate:"required"`
}

// Retrieves an inventory movement.
type RetrieveInventoryMovementEndpoint struct{}

func (e *RetrieveInventoryMovementEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveInventoryMovementRequest, *apiresource.InventoryMovement] {
	return (&apiendpoint.APIEndpoint[*RetrieveInventoryMovementRequest, *apiresource.InventoryMovement]{
		Title:             "Retrieve Inventory Movement",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             movementsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeInventoryMovement,
		Auth:              readMovements,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveInventoryMovementRequest) (*apiresource.InventoryMovement, *apierror.APIError) {
			return svc.(InventoryMovementSvc).GetInventoryMovement
		},
		IncludeConfig: movementIncludes(),
	})
}

// Request to create an inventory movement.
type CreateInventoryMovementRequest struct {
	// ID of the item whose stock changes. Services aren't stocked.
	ItemID string `json:"item_id" validate:"required"`
	// How `quantity` is applied.
	//
	// - `adjust`: adds `quantity` to the stock (negative to remove). Recorded as an `adjustment`.
	// - `reconcile`: sets the stock to exactly `quantity`, a count. Recorded as a `reconciliation`.
	Operation constants.InventoryUpdateOperation `json:"operation" validate:"required"`
	// The amount to add, or the count to set.
	//
	// In the item's stocking unit or one of its order units. An adjustment can't be `0`; a count can't be negative.
	Quantity apirequest.QuantityInput `json:"quantity" validate:"required"`
	// ID of the location the stock is at. It can't be archived.
	LocationID field.Optional[string] `json:"location_id,omitzero" validate:"omitempty"`
	// ID of an existing lot of the item to record the movement in. Send this or `lot_number`, not both.
	LotID field.Optional[string] `json:"lot_id,omitzero" validate:"omitempty"`
	// Number of the lot to record the movement in, created for the item if it doesn't exist yet. Send this or `lot_id`, not both.
	LotNumber field.Optional[string] `json:"lot_number,omitzero" validate:"omitempty,max=255"`
	// ID of the customer that owns stock being added, such as customer-supplied material you hold.
	//
	// Your account stays the holder. It affects only stock being added; a count is still measured against your account's stock. Requires edit access to the customer.
	CustomerID field.Optional[string] `json:"customer_id,omitzero" validate:"omitempty"`
	// Why stock was corrected: `damaged`, `count_correction`, `found`, `scrap` or `other`.
	Reason field.Optional[constants.InventoryMovementReason] `json:"reason,omitzero"`
}

var sampleCreateInventoryMovementRequest = &CreateInventoryMovementRequest{
	ItemID:     apiresource.SampleItemID,
	Operation:  constants.InventoryUpdateOperationAdjust,
	Quantity:   apirequest.QuantityInput{Value: "-4", UnitID: apiresource.SampleUnitID},
	LocationID: field.Some(apiresource.SampleLocationID),
	Reason:     field.Some(constants.InventoryMovementReasonDamaged),
}

func (*CreateInventoryMovementRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateInventoryMovementRequest)
}

// Records a change to an item's stock.
//
// `adjust` adds `quantity` (negative to remove). `reconcile` sets stock to exactly `quantity`, measured against what's on hand net of uncovered demand. Movements can't be edited or deleted; correct a mistake with another movement.
type CreateInventoryMovementEndpoint struct{}

func (e *CreateInventoryMovementEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateInventoryMovementRequest, *apiresource.InventoryMovement] {
	return (&apiendpoint.APIEndpoint[*CreateInventoryMovementRequest, *apiresource.InventoryMovement]{
		Title:             "Create Inventory Movement",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             movementsRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeInventoryMovement,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateInventoryMovementRequest) (*apiresource.InventoryMovement, *apierror.APIError) {
			return svc.(InventoryMovementSvc).CreateInventoryMovement
		},
		IncludeConfig: movementIncludes(),
	})
}

// One movement of a bulk request.
type BulkInventoryMovementInput struct {
	// ID of the item. Send this or `sku`.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// SKU of the item, matched exactly. Send this or `item_id`.
	SKU field.Optional[string] `json:"sku,omitzero" validate:"omitempty"`
	// The amount to add, or the count to set, in the item's stocking unit or one of its order units.
	Quantity apirequest.QuantityInput `json:"quantity" validate:"required"`
	// ID of the location the stock is at. It can't be archived.
	LocationID field.Optional[string] `json:"location_id,omitzero" validate:"omitempty"`
	// Number of the lot to record the movement in, created for the item if it doesn't exist yet.
	LotNumber field.Optional[string] `json:"lot_number,omitzero" validate:"omitempty,max=255"`
	// Why stock was corrected: `damaged`, `count_correction`, `found`, `scrap` or `other`.
	Reason field.Optional[constants.InventoryMovementReason] `json:"reason,omitzero"`
}

// Request to create many inventory movements.
type BulkCreateInventoryMovementsRequest struct {
	// How every movement's `quantity` is applied: `adjust` adds it, `reconcile` sets stock to it.
	Operation constants.InventoryUpdateOperation `json:"operation" validate:"required"`
	// The movements, at most 1,000.
	Movements []BulkInventoryMovementInput `json:"movements" validate:"required,min=1,max=1000"`
}

var sampleBulkCreateInventoryMovementsRequest = &BulkCreateInventoryMovementsRequest{
	Operation: constants.InventoryUpdateOperationReconcile,
	Movements: []BulkInventoryMovementInput{{
		SKU:      field.Some(apiresource.SampleItemSKU),
		Quantity: apirequest.QuantityInput{Value: "120", UnitID: apiresource.SampleUnitID},
		Reason:   field.Some(constants.InventoryMovementReasonCountCorrection),
	}},
}

func (*BulkCreateInventoryMovementsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkCreateInventoryMovementsRequest)
}

// Records many changes to stock in the background, such as a stocktake.
//
// Returns an async job to poll. Each movement is reported in the job's `results`: `created` with the movement, or `failed` with the reason (an unknown SKU, a unit that isn't one of the item's), and the rest still apply. A SKU listed twice applies in order. At most 1,000 movements and an 8 MB body.
type BulkCreateInventoryMovementsEndpoint struct{}

func (e *BulkCreateInventoryMovementsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkCreateInventoryMovementsRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*BulkCreateInventoryMovementsRequest, *apiresource.AsyncJob]{
		Title:             "Bulk Create Inventory Movements",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             movementsRoute + "/actions/bulk-create",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeAsyncJob,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionCreate}}},
		Extras:            apiendpoint.APIEndpointExtras{MaxJSONBodyBytes: httptransport.MaxJSONBodyBytes},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkCreateInventoryMovementsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(InventoryMovementSvc).BulkCreateInventoryMovements
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string { return "/v1/core/async-jobs/" + resp.ID },
	})
}

// Selects the inventory movements an export writes.
type ExportInventoryMovementsRequest struct {
	// Only export movements of any of these items.
	ItemIDs []string `json:"item_ids,omitzero"`
	// Only export movements of any of these types.
	Types []constants.InventoryMovementType `json:"types,omitzero"`
	// Only export movements caused by any of these users, API keys, agents or devices.
	ActorIDs []string `json:"actor_ids,omitzero"`
	// Only export movements caused by any of these kinds of actor.
	ActorTypes []constants.ActorType `json:"actor_types,omitzero"`
	// Only export movements at any of these locations.
	LocationIDs []string `json:"location_ids,omitzero"`
	// Only export movements that occurred at or after this instant. Leave it out to export from the first movement.
	OccurredAfter field.Optional[time.Time] `json:"occurred_after,omitzero"`
	// Only export movements that occurred at or before this instant.
	OccurredBefore field.Optional[time.Time] `json:"occurred_before,omitzero"`
}

var sampleExportInventoryMovementsRequest = &ExportInventoryMovementsRequest{
	Types:          []constants.InventoryMovementType{constants.InventoryMovementTypeAdjustment},
	OccurredAfter:  field.Some(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
	OccurredBefore: field.Some(time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)),
}

func (*ExportInventoryMovementsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleExportInventoryMovementsRequest)
}

// Exports inventory movements to an Excel file in the background.
//
// Returns an async job to poll; once it completes, `export.url` links to the file, one row per movement, newest first.
type ExportInventoryMovementsEndpoint struct{}

func (e *ExportInventoryMovementsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportInventoryMovementsRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ExportInventoryMovementsRequest, *apiresource.AsyncJob]{
		Title:             "Export Inventory Movements",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             movementsRoute + "/actions/export",
		SDKMethodKey:      "export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeAsyncJob,
		Auth:              readMovements,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportInventoryMovementsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(InventoryMovementSvc).ExportInventoryMovements
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string { return "/v1/core/async-jobs/" + resp.ID },
	})
}

// Request to list inventory levels.
type ListInventoryLevelsRequest struct {
	object.ListRequest
	// Only return levels of any of these items.
	ItemIDs []string `query:"item_ids"`
	// Count only stock at these locations. A level names its location when exactly one is given.
	LocationIDs []string `query:"location_ids"`
	// Count only stock at locations in these sites.
	SiteIDs []string `query:"site_ids"`
	// Report each item's stock as last logged at or before this instant instead of now.
	//
	// Only `on_hand` is known for a past instant: the item's physical stock (on hand less what was short) when its last movement by then was recorded. Can't be combined with `location_ids` or `site_ids`.
	AsOf *time.Time `query:"as_of"`
}

// Returns a list of inventory levels, one per stocked item.
//
// Every material, part and product is listed whether or not it holds stock, sorted by item creation date, newest first. Quantities are in each item's stocking unit.
type ListInventoryLevelsEndpoint struct{}

func (e *ListInventoryLevelsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListInventoryLevelsRequest, *apiresource.List[apiresource.InventoryLevel]] {
	return (&apiendpoint.APIEndpoint[*ListInventoryLevelsRequest, *apiresource.List[apiresource.InventoryLevel]]{
		Title:             "List Inventory Levels",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/inventory-levels",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeInventoryLevel,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListInventoryLevelsRequest) (*apiresource.List[apiresource.InventoryLevel], *apierror.APIError) {
			return svc.(InventoryMovementSvc).ListInventoryLevels
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInventoryLevel,
			Fields:     []string{"item", "location"},
		}),
	})
}

// Request to retrieve an item's inventory level.
type RetrieveItemInventoryLevelRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Retrieves an item's inventory level.
//
// The item's stock across every location, in its stocking unit. Services aren't stocked and have none.
type RetrieveItemInventoryLevelEndpoint struct{}

func (e *RetrieveItemInventoryLevelEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveItemInventoryLevelRequest, *apiresource.InventoryLevel] {
	return (&apiendpoint.APIEndpoint[*RetrieveItemInventoryLevelRequest, *apiresource.InventoryLevel]{
		Title:             "Retrieve Item Inventory Level",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}/inventory",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeInventoryLevel,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveItemInventoryLevelRequest) (*apiresource.InventoryLevel, *apierror.APIError) {
			return svc.(InventoryMovementSvc).GetItemInventoryLevel
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeInventoryLevel,
			Fields:     []string{"item", "location"},
		}),
	})
}

// Request to retrieve an item's lot size.
type RetrieveLotSizeDefaultRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Retrieves the quantity an item is made in.
//
// Resolved most specific first: a lot size set for the item in planning, then its item category's `default_lot_size`, then the account default. `lot_size` is `null` when none applies.
type RetrieveLotSizeDefaultEndpoint struct{}

func (e *RetrieveLotSizeDefaultEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveLotSizeDefaultRequest, *apiresource.LotSizeDefault] {
	return (&apiendpoint.APIEndpoint[*RetrieveLotSizeDefaultRequest, *apiresource.LotSizeDefault]{
		Title:             "Retrieve Item Lot Size Default",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}/lot-size-default",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeLotSizeDefault,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveLotSizeDefaultRequest) (*apiresource.LotSizeDefault, *apierror.APIError) {
			return svc.(InventoryMovementSvc).GetLotSizeDefault
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLotSizeDefault,
			Fields:     []string{"item", "item_category"},
		}),
	})
}
