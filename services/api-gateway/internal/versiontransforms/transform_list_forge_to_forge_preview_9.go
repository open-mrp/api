package versiontransforms

import (
	"net/url"
	"strconv"

	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge dropped page_info's has_next_page and has_prev_page, since another page exists exactly when its URL is
// present, and lowered the page size to 25 by default and 100 at most. preview.9 callers get the booleans back,
// derived from the URLs, and a larger limit they send is read as 100 rather than refused; they page on as before.
type listForgeToPreview9 struct {
	forgeToPreview9
	everyObject
	noRequestChange
}

func init() {
	version.RegisterTransformer(listForgeToPreview9{})
}

func (listForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		info, ok := obj["page_info"].(map[string]any)
		if !ok {
			return
		}
		info["has_next_page"] = info["next_page_url"] != nil
		info["has_prev_page"] = info["previous_page_url"] != nil
	})
	return data
}

func (listForgeToPreview9) TransformQuery(_ object.Type, _ string, query url.Values) url.Values {
	if limit, err := strconv.Atoi(query.Get("limit")); err == nil && limit > 100 {
		query.Set("limit", "100")
	}
	return query
}
