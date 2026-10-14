package versiontransforms

import (
	"crypto/rand"
	"encoding/binary"
	"net/http"
	"net/url"
	"strings"

	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge made an attribute's property an expandable field, null unless included, and creates an attribute with no
// color as the neutral `default` swatch. preview.9 callers that include an item's attributes still get each
// attribute's property, and an attribute they create without a color still gets one of the nine named colors.
type attributeForgeToPreview9 struct {
	forgeToPreview9
	everyObject
}

func init() {
	version.RegisterTransformer(attributeForgeToPreview9{})
}

const propertyAttributesRoute = "/v1/catalog/properties/{property_id}/attributes"

// preview9AttributeColors are the named colors preview.9 picked from when a new attribute had none.
var preview9AttributeColors = []string{"blue", "brown", "gray", "green", "orange", "pink", "purple", "red", "yellow"}

func (attributeForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	return data
}

func (attributeForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

func (attributeForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	if method != http.MethodPost || route != propertyAttributesRoute {
		return data
	}
	if _, ok := data["color"]; !ok {
		data["color"] = randomPreview9AttributeColor()
	}
	return data
}

func randomPreview9AttributeColor() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return preview9AttributeColors[binary.LittleEndian.Uint64(b[:])%uint64(len(preview9AttributeColors))]
}

// TransformQuery adds `<path>.attributes.property` beside every `<path>.attributes` the route also offers it for.
func (attributeForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	for _, param := range []string{"include", "include[]"} {
		values, ok := query[param]
		if !ok {
			continue
		}
		var keys []string
		seen := map[string]bool{}
		for _, v := range values {
			for _, key := range strings.Split(v, ",") {
				if key = strings.TrimSpace(key); key != "" && !seen[key] {
					seen[key] = true
					keys = append(keys, key)
				}
			}
		}
		for _, key := range keys {
			if key != "attributes" && !strings.HasSuffix(key, ".attributes") {
				continue
			}
			withProperty := key + ".property"
			if !seen[withProperty] && routeOffers(route, withProperty) {
				seen[withProperty] = true
				keys = append(keys, withProperty)
			}
		}
		if param == "include" {
			query[param] = []string{strings.Join(keys, ",")}
		} else {
			query[param] = keys
		}
	}
	return query
}
