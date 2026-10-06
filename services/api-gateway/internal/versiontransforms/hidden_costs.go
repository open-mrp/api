package versiontransforms

// zeroHiddenCosts answers 0 for each key of obj that holds null.
//
// preview.6 nulls a cost the caller may not read on fields preview.5 typed as plain numbers. A preview.5 caller cannot be handed a null its contract never allowed, and must not be handed the cost, so it reads 0. These fields are always set for a caller who may read costs, so a null on them only ever means the value was withheld.
func zeroHiddenCosts(obj map[string]any, keys ...string) {
	for _, key := range keys {
		if v, ok := obj[key]; ok && v == nil {
			obj[key] = float64(0)
		}
	}
}

// zeroHiddenCostsIn applies zeroHiddenCosts to every object in node whose `object` names an entry of keysByObject: a bare resource, a list envelope's rows, and one nested inside another resource.
func zeroHiddenCostsIn(node any, keysByObject map[string][]string) {
	switch v := node.(type) {
	case map[string]any:
		if keys, ok := keysByObject[asString(v["object"])]; ok {
			zeroHiddenCosts(v, keys...)
		}
		for _, child := range v {
			zeroHiddenCostsIn(child, keysByObject)
		}
	case []any:
		for _, child := range v {
			zeroHiddenCostsIn(child, keysByObject)
		}
	}
}
