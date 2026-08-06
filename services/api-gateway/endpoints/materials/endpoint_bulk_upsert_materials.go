package materialep

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

// Property name + value pair attached to a material. The property and its value (an
// attribute) are created if they do not yet exist.
type UpsertMaterialProperty struct {
	// Property name (e.g. "Grade"). Matched case-insensitively; created if missing.
	Name string `json:"name" validate:"required,max=255"`
	// Property value (e.g. "A36"). Matched case-insensitively; created under the property
	// if missing. A value already in use under a different property fails the whole job.
	Value string `json:"value" validate:"required,max=255"`
}

// Input for a single material in a bulk upsert operation.
type UpsertMaterialInput struct {
	// SKU for the material, used to match an existing material within the account. If it
	// exists the material is updated in place; otherwise a new material is created. A SKU
	// already used by a non-material item fails that row.
	SKU string `json:"sku" validate:"required,max=255"`
	// Material description.
	Description field.Optional[string] `json:"description,omitzero"`
	// Material notes.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Item category to place the material in, referenced by `id` or `name`. Create-only.
	Category apirequest.ObjectIdentifier `json:"category" validate:"required"`
	// Reorder threshold quantity. When omitted on create it defaults to a zero quantity in
	// the category's base unit.
	OrderPoint field.Optional[QuantityInputRequest] `json:"order_point,omitzero"`
	// Expected lead time quantity. When omitted on create it defaults to a zero quantity in
	// the category's base unit.
	LeadTime field.Optional[QuantityInputRequest] `json:"lead_time,omitzero"`
	// Selling price per unit. Numerator must be a currency unit, denominator the per-unit
	// basis. Defaults to a zero rate in the category's base unit on create; unchanged when
	// omitted on update.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Cost per unit. Same currency-vs-non-currency rule as `unit_price`.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero"`
	// Properties to attach to the material, matched/created by name + value. Additive —
	// existing attributes are not removed.
	Properties []UpsertMaterialProperty `json:"properties" default:"[]" validate:"dive"`
}

// Request to bulk upsert materials.
type BulkUpsertMaterialsRequest struct {
	// Materials to create or update, matched by SKU within the account.
	Materials []UpsertMaterialInput `json:"materials" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertMaterialsRequest = &BulkUpsertMaterialsRequest{
	Materials: []UpsertMaterialInput{
		{
			SKU:      "MAT-001",
			Category: apirequest.ObjectIdentifier{ID: apiresource.SampleItemCategoryID},
		},
	},
}

func (*BulkUpsertMaterialsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertMaterialsRequest)
}

// Creates or updates multiple materials for the account, matched by SKU. Validates and
// resolves synchronously, then writes asynchronously — 202 with a job to poll.
type BulkUpsertMaterialsEndpoint struct{}

func (e *BulkUpsertMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertMaterialsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertMaterialsRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Materials",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/materials/actions/bulk-upsert",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertMaterialsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(MaterialSvc).BulkUpsertMaterials
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
