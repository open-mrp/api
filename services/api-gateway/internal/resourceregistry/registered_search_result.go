package resourceregistry

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/include"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

// A search result's record and entity are loaded by the search itself, which knows each match's type, and stashed
// under "<type>:<id>" (see searchep.SearchResultMetaKey).
func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeSearchResult,
		Load: func(context.Context, []string) (map[string]any, *apierror.APIError) {
			return nil, apierror.NewInvariantViolationError("search results are never loaded by ID")
		},
		Subs: []include.SubField{
			{Key: "resource", Populate: func(ctx context.Context, parent any, _ map[string]any) {
				r := parent.(*apiresource.SearchResult)
				if v, ok := searchResultMeta(ctx, r, "resource"); ok {
					r.Resource = v.(json.RawMessage)
				}
			}},
			{Key: "entity", Populate: func(ctx context.Context, parent any, _ map[string]any) {
				r := parent.(*apiresource.SearchResult)
				if v, ok := searchResultMeta(ctx, r, "entity"); ok {
					r.Entity = v.(*apiresource.Entity)
				}
			}},
		},
	})
}

func searchResultMeta(ctx context.Context, r *apiresource.SearchResult, key string) (any, bool) {
	return include.GetLoadMeta(ctx).Get(constants.ObjectTypeSearchResult, string(r.Type)+":"+r.ResourceID, key)
}
