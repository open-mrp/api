package apiresource

import (
	"encoding/json"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
)

// One record a search matched.
type SearchResult struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=search_result"`
	// The kind of record matched: `sales_order`, `purchase_order`, `invoice`, `customer`, `item`, `product`, `shipment`, `account_user` or `agent_definition`.
	Type constants.ObjectType `json:"type" validate:"required"`
	// ID of the matched record.
	ResourceID string `json:"resource_id" validate:"required"`
	// The matched record, in the shape its own retrieve endpoint returns, without its own expandable fields.
	Resource json.RawMessage `json:"resource" expandable:"true"`
	// Kept for 1.0.forge-preview.9 callers, which read results as entity references.
	Entity *Entity `json:"entity" expandable:"true" visibility:"internal"`
}

var SampleSearchResult = &SearchResult{
	Object:     constants.ObjectTypeSearchResult,
	Type:       constants.ObjectTypeCustomer,
	ResourceID: SampleCustomerID,
	Resource:   mustMarshalSample(SampleCustomer),
}

func (*SearchResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSearchResult)
}

func mustMarshalSample(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}
