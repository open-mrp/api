package versiontransforms

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge renamed an item category's type from material_category | product_category to material | product, removed its
// notes, and filters the list by types[] instead of type. preview.9 callers see the old type names and a null notes
// wherever a category appears; their writes have the type renamed and notes dropped (notes are no longer stored on
// the category), and their type filter becomes types[].
type itemCategoryForgeToPreview9 struct {
	forgeToPreview9
	everyObject
}

func init() {
	version.RegisterTransformer(itemCategoryForgeToPreview9{})
}

const (
	itemCategoriesRoute           = "/v1/catalog/item-categories"
	itemCategoryRoute             = "/v1/catalog/item-categories/{id}"
	itemCategoriesBulkUpsertRoute = "/v1/catalog/item-categories/actions/bulk-upsert"
	preview9ItemCategorySuffix    = "_category"
)

func (itemCategoryForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] != "item_category" {
			return
		}
		if t, ok := obj["type"].(string); ok && t != "" && !strings.HasSuffix(t, preview9ItemCategorySuffix) {
			obj["type"] = t + preview9ItemCategorySuffix
		}
		if _, ok := obj["notes"]; !ok {
			obj["notes"] = nil
		}
	})
	return data
}

func (itemCategoryForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

func (itemCategoryForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	switch {
	case method == http.MethodPost && route == itemCategoriesRoute:
		upgradeItemCategoryWrite(data)
	case method == http.MethodPatch && route == itemCategoryRoute:
		upgradeItemCategoryWrite(data)
	case method == http.MethodPost && route == itemCategoriesBulkUpsertRoute:
		rows, _ := data["item_categories"].([]any)
		for _, row := range rows {
			if m, ok := row.(map[string]any); ok {
				upgradeItemCategoryWrite(m)
			}
		}
	}
	return data
}

// upgradeItemCategoryWrite renames a preview.9 type and drops notes, which forge no longer accepts.
func upgradeItemCategoryWrite(body map[string]any) {
	if t, ok := body["type"].(string); ok {
		body["type"] = strings.TrimSuffix(t, preview9ItemCategorySuffix)
	}
	delete(body, "notes")
}

func (itemCategoryForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	if route != itemCategoriesRoute {
		return query
	}
	if t := query.Get("type"); t != "" {
		query.Del("type")
		query.Add("types[]", strings.TrimSuffix(t, preview9ItemCategorySuffix))
	}
	return query
}
