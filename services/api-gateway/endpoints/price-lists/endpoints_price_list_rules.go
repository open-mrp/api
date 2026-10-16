package pricelistep

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

func ruleIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{ObjectType: constants.ObjectTypePriceListRule, Fields: ruleIncludes})
}

// The items a rule prices. Leave a list empty to match any item; an item must match every list that isn't empty.
type PriceListRuleScopeInput struct {
	// IDs of the items the rule prices.
	ItemIDs []string `json:"item_ids,omitzero" validate:"omitempty,max=100"`
	// IDs of the product lines whose items the rule prices.
	ProductLineIDs []string `json:"product_line_ids,omitzero" validate:"omitempty,max=100"`
	// IDs of the item categories whose items the rule prices.
	CategoryIDs []string `json:"category_ids,omitzero" validate:"omitempty,max=100"`
	// IDs of attributes an item must all have for the rule to price it.
	AttributeIDs []string `json:"attribute_ids,omitzero" validate:"omitempty,max=100"`
}

// Scope fields to replace. Each list you send replaces that list; send `[]` to match any item on it.
type UpdatePriceListRuleScopeInput struct {
	// IDs of the items the rule prices.
	ItemIDs field.Optional[[]string] `json:"item_ids,omitzero" validate:"omitempty,max=100"`
	// IDs of the product lines whose items the rule prices.
	ProductLineIDs field.Optional[[]string] `json:"product_line_ids,omitzero" validate:"omitempty,max=100"`
	// IDs of the item categories whose items the rule prices.
	CategoryIDs field.Optional[[]string] `json:"category_ids,omitzero" validate:"omitempty,max=100"`
	// IDs of attributes an item must all have for the rule to price it.
	AttributeIDs field.Optional[[]string] `json:"attribute_ids,omitzero" validate:"omitempty,max=100"`
}

// The price a rule sets.
type PriceListPriceInput struct {
	// How the rule sets the price.
	//
	// - `fixed`: exactly `unit_price`.
	// - `percent_of_base`: `percent_of_base` percent of the item's `sales.unit_price`.
	// - `tiered`: the highest of `tiers` reached by the quantity ordered across every line in the rule's scope, counted in `unit_id`.
	Type constants.PriceListPriceType `json:"type" validate:"required"`
	// The price per unit. Required for `fixed`.
	UnitPrice field.Optional[apirequest.UnitPriceInput] `json:"unit_price,omitzero"`
	// Percent of the item's base sales price, greater than 0, with up to 6 decimal places: `"80"` is 20% off, `"120"` a 20% markup. Required for `percent_of_base`.
	PercentOfBase field.Optional[string] `json:"percent_of_base,omitzero" format:"decimal"`
	// ID of the unit quantities are counted in. Required for `tiered`.
	UnitID field.Optional[string] `json:"unit_id,omitzero"`
	// The quantity breaks, 1 to 20, for `tiered`. Sent in any order; each `min_quantity` must differ.
	Tiers []PriceListTierInput `json:"tiers,omitzero" validate:"omitempty,max=20,dive"`
}

// One quantity break of a tiered price. Set exactly one of `percent_of_base` or `unit_price`.
type PriceListTierInput struct {
	// The quantity, counted in the rule's unit, from which this tier applies. 0 or more.
	MinQuantity string `json:"min_quantity" validate:"required" format:"decimal"`
	// Percent of the item's base sales price, greater than 0.
	PercentOfBase field.Optional[string] `json:"percent_of_base,omitzero" format:"decimal"`
	// The price per unit.
	UnitPrice field.Optional[apirequest.UnitPriceInput] `json:"unit_price,omitzero"`
}

// Request to list a price list's rules.
type ListPriceListRulesRequest struct {
	object.ListRequest
	// Price list ID.
	PriceListID string `path:"price_list_id" validate:"required"`
	// Only return rules whose scope names one of these items.
	ItemIDs []string `query:"item_ids" validate:"omitempty,max=100"`
	// Only return rules whose scope names one of these product lines.
	ProductLineIDs []string `query:"product_line_ids" validate:"omitempty,max=100"`
}

// Returns a list of a price list's rules.
//
// Returns the list's rules, newest first.
type ListPriceListRulesEndpoint struct{}

func (e *ListPriceListRulesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPriceListRulesRequest, *apiresource.List[apiresource.PriceListRule]] {
	return (&apiendpoint.APIEndpoint[*ListPriceListRulesRequest, *apiresource.List[apiresource.PriceListRule]]{
		Title:             "List Price List Rules",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             rulesRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePriceListRule,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead}}},
		IncludeConfig:     ruleIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPriceListRulesRequest) (*apiresource.List[apiresource.PriceListRule], *apierror.APIError) {
			return svc.(PriceListSvc).ListRules
		},
	})
}

// Request to retrieve a price list rule.
type RetrievePriceListRuleRequest struct {
	// Price list ID.
	PriceListID string `path:"price_list_id" validate:"required"`
	// Price list rule ID.
	ID string `path:"id" validate:"required"`
}

// Retrieves a price list rule.
type RetrievePriceListRuleEndpoint struct{}

func (e *RetrievePriceListRuleEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePriceListRuleRequest, *apiresource.PriceListRule] {
	return (&apiendpoint.APIEndpoint[*RetrievePriceListRuleRequest, *apiresource.PriceListRule]{
		Title:             "Retrieve Price List Rule",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             ruleRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePriceListRule,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead}}},
		IncludeConfig:     ruleIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePriceListRuleRequest) (*apiresource.PriceListRule, *apierror.APIError) {
			return svc.(PriceListSvc).GetRule
		},
	})
}

// Request to create a price list rule.
type CreatePriceListRuleRequest struct {
	// Price list ID.
	PriceListID string `path:"price_list_id" validate:"required"`
	// The items the rule prices. Omit to price any item.
	Scope field.Optional[PriceListRuleScopeInput] `json:"scope,omitzero"`
	// The price the rule sets.
	Price PriceListPriceInput `json:"price" validate:"required"`
}

var sampleCreatePriceListRuleRequest = &CreatePriceListRuleRequest{
	Scope: field.Some(sampleRuleScope),
	Price: PriceListPriceInput{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: field.Some("80")},
}

func (*CreatePriceListRuleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePriceListRuleRequest)
}

// Creates a price list rule.
//
// A line is priced by the most specific matching rule across the lists that apply to its customer: an item scope beats attributes or a category, which beat a product line, which beats an empty scope. Ties go to the customer's own lists, then its parent's, then its group's, then the newer rule.
type CreatePriceListRuleEndpoint struct{}

func (e *CreatePriceListRuleEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePriceListRuleRequest, *apiresource.PriceListRule] {
	return (&apiendpoint.APIEndpoint[*CreatePriceListRuleRequest, *apiresource.PriceListRule]{
		Title:             "Create Price List Rule",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             rulesRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePriceListRule,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionCreate}}},
		IncludeConfig:     ruleIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePriceListRuleRequest) (*apiresource.PriceListRule, *apierror.APIError) {
			return svc.(PriceListSvc).CreateRule
		},
	})
}

// Request to update a price list rule.
type UpdatePriceListRuleRequest struct {
	// Price list ID.
	PriceListID string `path:"price_list_id" validate:"required"`
	// Price list rule ID.
	ID string `path:"id" validate:"required"`
	// Scope lists to replace.
	Scope field.Optional[UpdatePriceListRuleScopeInput] `json:"scope,omitzero"`
	// The price the rule sets. Replaces the whole price.
	Price field.Optional[PriceListPriceInput] `json:"price,omitzero"`
}

var sampleUpdatePriceListRuleRequest = &UpdatePriceListRuleRequest{
	Price: field.Some(PriceListPriceInput{Type: constants.PriceListPriceTypePercentOfBase, PercentOfBase: field.Some("85")}),
}

func (*UpdatePriceListRuleRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePriceListRuleRequest)
}

// Updates a price list rule.
//
// Only fields you send are changed. Orders already priced keep their prices.
type UpdatePriceListRuleEndpoint struct{}

func (e *UpdatePriceListRuleEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePriceListRuleRequest, *apiresource.PriceListRule] {
	return (&apiendpoint.APIEndpoint[*UpdatePriceListRuleRequest, *apiresource.PriceListRule]{
		Title:             "Update Price List Rule",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             ruleRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePriceListRule,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionUpdate}}},
		IncludeConfig:     ruleIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePriceListRuleRequest) (*apiresource.PriceListRule, *apierror.APIError) {
			return svc.(PriceListSvc).UpdateRule
		},
	})
}

// Request to delete a price list rule.
type DeletePriceListRuleRequest struct {
	// Price list ID.
	PriceListID string `path:"price_list_id" validate:"required"`
	// Price list rule ID.
	ID string `path:"id" validate:"required"`
}

// Deletes a price list rule.
//
// Orders already priced by it keep their prices.
type DeletePriceListRuleEndpoint struct{}

func (e *DeletePriceListRuleEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePriceListRuleRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeletePriceListRuleRequest, *object.Deleted]{
		Title:             "Delete Price List Rule",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             ruleRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePriceListRule,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePriceListRuleRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(PriceListSvc).DeleteRule, constants.ObjectTypePriceListRule, "id")
		},
	})
}
