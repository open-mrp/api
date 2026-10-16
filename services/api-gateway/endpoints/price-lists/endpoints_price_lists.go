package pricelistep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

func priceListIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{ObjectType: constants.ObjectTypePriceList, Fields: priceListIncludes})
}

// Request to list price lists.
type ListPriceListsRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return lists with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
	// Only return lists that apply to one of these customers: assigned to the customer, its parent customer or its account group.
	CustomerIDs []string `query:"customer_ids" validate:"omitempty,max=100"`
}

// Returns a list of price lists.
//
// Returns your account's price lists, newest first.
type ListPriceListsEndpoint struct{}

func (e *ListPriceListsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPriceListsRequest, *apiresource.List[apiresource.PriceList]] {
	return (&apiendpoint.APIEndpoint[*ListPriceListsRequest, *apiresource.List[apiresource.PriceList]]{
		Title:             "List Price Lists",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             priceListsRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePriceList,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead}}},
		IncludeConfig:     priceListIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPriceListsRequest) (*apiresource.List[apiresource.PriceList], *apierror.APIError) {
			return svc.(PriceListSvc).ListPriceLists
		},
	})
}

// Request to retrieve a price list.
type RetrievePriceListRequest struct {
	// Price list ID.
	ID string `path:"id" validate:"required"`
}

// Retrieves a price list.
type RetrievePriceListEndpoint struct{}

func (e *RetrievePriceListEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePriceListRequest, *apiresource.PriceList] {
	return (&apiendpoint.APIEndpoint[*RetrievePriceListRequest, *apiresource.PriceList]{
		Title:             "Retrieve Price List",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             priceListRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePriceList,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead}}},
		IncludeConfig:     priceListIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePriceListRequest) (*apiresource.PriceList, *apierror.APIError) {
			return svc.(PriceListSvc).GetPriceList
		},
	})
}

// Request to create a price list.
type CreatePriceListRequest struct {
	// Name of the list, such as `Distributor pricing`.
	Name string `json:"name" validate:"required,max=255"`
	// What the list is for.
	Description field.Optional[string] `json:"description,omitzero" validate:"omitempty,max=5000"`
	// Whether the list's rules apply. Defaults to `active`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Three-letter ISO 4217 currency code of the list's prices, lowercase. Only `usd` is supported; any other code fails with `currency_not_supported`.
	Currency field.Optional[string] `json:"currency,omitzero"`
}

var sampleCreatePriceListRequest = &CreatePriceListRequest{
	Name:        "Distributor pricing",
	Description: field.Some("Prices for distributors buying by the carton."),
}

func (*CreatePriceListRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePriceListRequest)
}

// Creates a price list.
//
// A new list has no rules and applies to no one: add rules, then assign it with a customer's or account group's `price_list_ids`.
type CreatePriceListEndpoint struct{}

func (e *CreatePriceListEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePriceListRequest, *apiresource.PriceList] {
	return (&apiendpoint.APIEndpoint[*CreatePriceListRequest, *apiresource.PriceList]{
		Title:             "Create Price List",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             priceListsRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePriceList,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionCreate}}},
		IncludeConfig:     priceListIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePriceListRequest) (*apiresource.PriceList, *apierror.APIError) {
			return svc.(PriceListSvc).CreatePriceList
		},
		LocationFunc: func(resp *apiresource.PriceList) string { return priceListsRoute + "/" + resp.ID },
	})
}

// Request to update a price list.
type UpdatePriceListRequest struct {
	// Price list ID.
	ID string `path:"id" validate:"required"`
	// Name of the list.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// What the list is for. Send `null` to clear it.
	Description field.Clearable[string] `json:"description,omitzero" validate:"omitempty,max=5000"`
	// Whether the list's rules apply. `archived` lists stop applying to new orders immediately.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
}

var sampleUpdatePriceListRequest = &UpdatePriceListRequest{
	Status: field.Some(constants.ArchiveStatusArchived),
}

func (*UpdatePriceListRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePriceListRequest)
}

// Updates a price list.
//
// Only fields you send are changed. Archive a list by setting `status` to `archived`: its rules stop applying to new orders immediately.
type UpdatePriceListEndpoint struct{}

func (e *UpdatePriceListEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePriceListRequest, *apiresource.PriceList] {
	return (&apiendpoint.APIEndpoint[*UpdatePriceListRequest, *apiresource.PriceList]{
		Title:             "Update Price List",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             priceListRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePriceList,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionUpdate}}},
		IncludeConfig:     priceListIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePriceListRequest) (*apiresource.PriceList, *apierror.APIError) {
			return svc.(PriceListSvc).UpdatePriceList
		},
	})
}

// Request to delete a price list.
type DeletePriceListRequest struct {
	// Price list ID.
	ID string `path:"id" validate:"required"`
}

// Deletes a price list.
//
// Only while no customer or account group is assigned it; its rules are deleted with it. Archive a list that is in use by setting `status` to `archived`.
type DeletePriceListEndpoint struct{}

func (e *DeletePriceListEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePriceListRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeletePriceListRequest, *object.Deleted]{
		Title:             "Delete Price List",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             priceListRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePriceList,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePriceListRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(PriceListSvc).DeletePriceList, constants.ObjectTypePriceList, "id")
		},
	})
}
