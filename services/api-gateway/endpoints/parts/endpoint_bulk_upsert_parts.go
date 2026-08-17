package partep

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

// UpsertPartProperty is a property name + value pair attached to a part. The property
// and its value (an attribute) are created if they do not yet exist.
type UpsertPartProperty struct {
	// Property name (e.g. "Material"). Matched case-insensitively; created if missing.
	Name string `json:"name" validate:"required,max=255"`
	// Property value (e.g. "Steel"). Matched exactly; created under the property if missing.
	Value string `json:"value" validate:"required,max=255" format:"decimal"`
}

// UpsertPartInput is the input for a single part in a bulk upsert operation.
type UpsertPartInput struct {
	// SKU for the part, matched against existing parts in the account: a match updates in
	// place, otherwise a part is created. A SKU held by a non-part item fails that row.
	SKU string `json:"sku" validate:"required,max=255"`
	// Free-form description of the part.
	Description field.Optional[string] `json:"description,omitzero"`
	// Free-form notes about the part.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// Item category to place the part in, referenced by `id` or `name`. Create-only; its
	// unit group determines the base unit of the part's rates.
	Category apirequest.ObjectIdentifier `json:"category" validate:"required"`
	// Selling price per unit — a currency numerator over a per-unit denominator. Omitted, it
	// defaults to a zero rate in the category's base unit and is left unchanged on update.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Cost per unit. Same unit rule and omission behaviour as `unit_price`.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero"`
	// Properties to attach to the part, matched/created by name + value. Additive —
	// existing attributes are not removed.
	Properties []UpsertPartProperty `json:"properties" default:"[]" validate:"dive"`
}

// BulkUpsertPartsRequest is the request to bulk upsert parts.
type BulkUpsertPartsRequest struct {
	// Parts to create or update, matched by SKU within the account.
	Parts []UpsertPartInput `json:"parts" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertPartsRequest = &BulkUpsertPartsRequest{
	Parts: []UpsertPartInput{
		{
			SKU:      apiresource.SamplePartSKU,
			Category: apirequest.ObjectIdentifier{ID: apiresource.SampleItemCategoryID},
		},
	},
}

func (*BulkUpsertPartsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertPartsRequest)
}

// Creates or updates multiple parts for the account, matched by SKU, then writes
// asynchronously — 202 with a job to poll.
type BulkUpsertPartsEndpoint struct{}

func (e *BulkUpsertPartsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertPartsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertPartsRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Parts",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/parts/actions/bulk-upsert",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertPartsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(PartSvc).BulkUpsertParts
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
