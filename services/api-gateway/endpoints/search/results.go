package searchep

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// searchResourceLoaders load the full record behind each searchable type, for the `resource` include.
var searchResourceLoaders = map[constants.ObjectType]func(ctx context.Context, ids []string) (map[string]any, *apierror.APIError){
	constants.ObjectTypeSalesOrder:      resourceloaders.LoadSalesOrders,
	constants.ObjectTypePurchaseOrder:   resourceloaders.LoadPurchaseOrders,
	constants.ObjectTypeInvoice:         resourceloaders.LoadInvoices,
	constants.ObjectTypeCustomer:        resourceloaders.LoadCustomers,
	constants.ObjectTypeItem:            resourceloaders.LoadItems,
	constants.ObjectTypeProduct:         resourceloaders.LoadProducts,
	constants.ObjectTypeShipment:        resourceloaders.LoadShipments,
	constants.ObjectTypeAccountUser:     resourceloaders.LoadAccountUsers,
	constants.ObjectTypeAgentDefinition: resourceloaders.LoadAgentDefinitions,
}

// SearchResultMetaKey is the LoadMeta key a search result's include data is stashed under: its type and record ID.
func SearchResultMetaKey(t constants.ObjectType, id string) string {
	return string(t) + ":" + id
}

// searchResults turns the matches into results. Each match's entity reference is stashed for the internal `entity`
// include, and when `resource` is requested the matched records are loaded, one batch per type, and stashed too.
func searchResults(ctx context.Context, matches []apiresource.Entity) []apiresource.SearchResult {
	meta := include.GetLoadMeta(ctx)
	out := make([]apiresource.SearchResult, len(matches))
	byType := map[constants.ObjectType][]string{}
	for i, m := range matches {
		out[i] = apiresource.SearchResult{Object: constants.ObjectTypeSearchResult, Type: m.Type, ResourceID: m.ID}
		key := SearchResultMetaKey(m.Type, m.ID)
		meta.Set(constants.ObjectTypeSearchResult, key, "entity", new(m))
		byType[m.Type] = append(byType[m.Type], m.ID)
	}
	if !include.RequestedIncludeSet(ctx)["resource"] {
		return out
	}

	var wg sync.WaitGroup
	for t, ids := range byType {
		load, ok := searchResourceLoaders[t]
		if !ok {
			continue
		}
		wg.Go(func() {
			loaded, apiErr := load(ctx, ids)
			if apiErr != nil {
				slog.WarnContext(ctx, "search resource load failed", "type", string(t), "error", apiErr.Error())
				return
			}
			for id, v := range loaded {
				raw, err := json.Marshal(v)
				if err != nil {
					continue
				}
				meta.Set(constants.ObjectTypeSearchResult, SearchResultMetaKey(t, id), "resource", json.RawMessage(raw))
			}
		})
	}
	wg.Wait()
	return out
}
