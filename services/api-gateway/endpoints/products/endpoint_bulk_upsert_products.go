package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Property name + value pair attached to a product. The property and its value (an
// attribute) are created if they do not yet exist.
type UpsertProductProperty struct {
	// Property name (e.g. "Color"). Matched case-insensitively; created if missing.
	Name string `json:"name" validate:"required,max=255"`
	// Property value (e.g. "Red"). Matched case-insensitively; created under the property
	// if missing. A value already in use under a different property fails the whole job.
	Value string `json:"value" validate:"required,max=255"`
}

// Input for a single product in a bulk upsert operation.
type UpsertProductInput struct {
	// SKU for the product, used to match an existing product within the account. If it
	// exists the product is updated in place; otherwise a new product is created. A SKU
	// already used by a non-product item fails that row.
	SKU string `json:"sku" validate:"required,max=255"`
	// Product type. Create-only; defaults to `sale` when omitted.
	Type field.Optional[constants.ProductTypeCode] `json:"type,omitzero"`
	// Product description.
	Description field.Optional[string] `json:"description,omitzero"`
	// Product notes.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Item category to place the product in, referenced by `id` or `name`. Create-only.
	Category apirequest.ObjectIdentifier `json:"category" validate:"required"`
	// Product line to assign, referenced by `id` or `name`. Create-only.
	ProductLine field.Optional[apirequest.ObjectIdentifier] `json:"product_line,omitzero"`
	// Whether the product is shown to buyers in the customer portal. Defaults to `hidden`
	// on create; preserved when omitted on update.
	PortalVisibility field.Optional[constants.CustomerPortalVisibility] `json:"portal_visibility,omitzero"`
	// Selling price per unit. Numerator must be a currency unit, denominator the per-unit
	// basis. Defaults to a zero rate in the category's base unit on create; unchanged when
	// omitted on update.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Cost per unit. Same currency-vs-non-currency rule as `unit_price`.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero"`
	// Properties to attach to the product, matched/created by name + value. Additive —
	// existing attributes are not removed.
	Properties []UpsertProductProperty `json:"properties" default:"[]" validate:"dive"`
}

// Request to bulk upsert products.
type BulkUpsertProductsRequest struct {
	// Products to create or update, matched by SKU within the account.
	Products []UpsertProductInput `json:"products" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertProductsRequest = &BulkUpsertProductsRequest{
	Products: []UpsertProductInput{
		{
			SKU:      apiresource.SampleItemSKU,
			Category: apirequest.ObjectIdentifier{ID: apiresource.SampleItemCategoryID},
		},
	},
}

func (*BulkUpsertProductsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertProductsRequest)
}

// Creates or updates multiple products for the account, matched by SKU. Validates and
// resolves synchronously, then writes asynchronously — 202 with a job to poll.
type BulkUpsertProductsEndpoint struct{}

func (e *BulkUpsertProductsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertProductsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertProductsRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Products",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/products/actions/bulk-upsert",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertProductsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ProductSvc).BulkUpsertProducts
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
