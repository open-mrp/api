package redact

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
)

// SensitiveFields collects dot-separated JSON field paths declared with sensitive:"true" (secrets) or sensitive:"cost" (the seller's cost data, which a log reader may not be allowed to see) on structs reachable from root type typ, and for a map tagged sensitive:"cost_keys" the path of its cost-named keys (CostKey). Root may be a pointer (e.g. *MyRequest); non-struct roots return nil.
//
// Embedding without a JSON key name preserves the same path prefix so promoted fields align with encoding/json flattened output.
func SensitiveFields(typ reflect.Type) map[string]bool {
	typ = deref(typ)
	if typ == nil || typ.Kind() != reflect.Struct {
		return nil
	}
	out := make(map[string]bool)
	collect(typ, "", out, 0)
	if len(out) == 0 {
		return nil
	}
	return out
}

// IsSensitiveTag reports whether a sensitive struct tag value keeps the field out of logs.
func IsSensitiveTag(tag string) bool {
	return tag == "true" || tag == "cost"
}

// TagCostKeys marks a map whose keys name what they hold, so the entries whose key reads as cost data (IsCostName) are the sensitive ones.
const TagCostKeys = "cost_keys"

var costNameTokens = map[string]bool{
	"cost": true, "costs": true, "cogs": true, "margin": true, "margins": true,
	"profit": true, "profits": true, "valuation": true, "markup": true,
}

var costNames = map[string]bool{
	"labor_rate":            true,
	"overhead_rate":         true,
	"changeover_labor_rate": true,
	"inventory_value":       true,
}

// IsCostName reports whether a snake_case field name reads as the seller's cost or margin data.
func IsCostName(name string) bool {
	if costNames[name] {
		return true
	}
	for token := range strings.SplitSeq(name, "_") {
		if costNameTokens[token] {
			return true
		}
	}
	return false
}

func deref(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

func parseJSONTagName(tag string) (name string, skip bool) {
	if tag == "" {
		return "", false
	}
	name = strings.TrimSpace(strings.Split(tag, ",")[0])
	if name == "-" {
		return "", true
	}
	return name, false
}

func pathJoin(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

const maxSensitiveFieldDepth = 32

func collect(typ reflect.Type, prefix string, out map[string]bool, depth int) {
	collectWithVisited(typ, prefix, out, depth, make(map[reflect.Type]bool))
}

func collectWithVisited(typ reflect.Type, prefix string, out map[string]bool, depth int, visited map[reflect.Type]bool) {
	typ = deref(typ)
	if typ == nil || typ.Kind() != reflect.Struct {
		return
	}
	if depth > maxSensitiveFieldDepth {
		return
	}
	if visited[typ] {
		return
	}
	visited[typ] = true
	defer func() { delete(visited, typ) }()

	for sf := range typ.Fields() {
		if !sf.IsExported() {
			continue
		}

		if sf.Anonymous {
			jsonName, skip := parseJSONTagName(sf.Tag.Get("json"))
			if skip {
				continue
			}
			if jsonName != "" {
				collectWithVisited(sf.Type, pathJoin(prefix, jsonName), out, depth+1, visited)
			} else {
				collectWithVisited(sf.Type, prefix, out, depth+1, visited)
			}
			continue
		}

		jsonName, skip := parseJSONTagName(sf.Tag.Get("json"))
		if skip || jsonName == "" {
			continue
		}

		path := pathJoin(prefix, jsonName)

		ft := sf.Type
		isSensitive := IsSensitiveTag(sf.Tag.Get("sensitive"))
		ftd := deref(ft)

		if isSensitive {
			out[path] = true
			continue
		}
		if sf.Tag.Get("sensitive") == TagCostKeys {
			out[pathJoin(path, CostKey)] = true
			continue
		}

		switch ftd.Kind() {
		case reflect.Struct:
			collectWithVisited(ft, path, out, depth+1, visited)
		case reflect.Slice, reflect.Array:
			elem := deref(ftd.Elem())
			if elem.Kind() == reflect.Struct {
				collectWithVisited(ftd.Elem(), path, out, depth+1, visited)
			}
		case reflect.Map:
			elem := deref(ftd.Elem())
			if elem.Kind() == reflect.Struct {
				collectWithVisited(ftd.Elem(), pathJoin(path, MapKey), out, depth+1, visited)
			}
		}
	}
}

// MapKey is the path segment standing for any key of a map, whose keys are data rather than field names.
const MapKey = "*"

// CostKey is the path segment standing for any key of a map that reads as cost data (IsCostName).
const CostKey = "*cost*"

// RedactJSON replaces JSON values whose paths exactly match sensitivePaths keys with the JSON string ****. Arrays reuse the parent's path segment so structs under an array resolve the same dotted paths encoding/json emits (no index in the path), and a MapKey segment matches any object key.
//
// On unmarshal marshal failure returns nil so callers omit the logged body entirely.
func RedactJSON(raw []byte, sensitivePaths map[string]bool) []byte {
	if len(sensitivePaths) == 0 {
		return slices.Clone(raw)
	}
	if len(raw) == 0 {
		return slices.Clone(raw)
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var root any
	if err := dec.Decode(&root); err != nil {
		return nil
	}

	redactAny(root, []string{""}, sensitivePaths, pathPrefixes(sensitivePaths))

	out, err := json.Marshal(root)
	if err != nil {
		return nil
	}
	return out
}

// redactAny walks v with every path it may be at: an object key can be a field name or a map key, so each step tries the key itself, MapKey and, for a key that reads as cost data, CostKey, keeping only paths that lead to a sensitive one.
func redactAny(v any, paths []string, sensitivePaths, prefixes map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			var next []string
			masked := false
			segs := []string{k, MapKey}
			if IsCostName(k) {
				segs = append(segs, CostKey)
			}
			for _, p := range paths {
				for _, seg := range segs {
					cur := pathJoin(p, seg)
					if sensitivePaths[cur] {
						masked = true
					} else if prefixes[cur] {
						next = append(next, cur)
					}
				}
			}
			if masked {
				x[k] = "****"
				continue
			}
			if len(next) > 0 {
				redactAny(child, next, sensitivePaths, prefixes)
			}
		}
	case []any:
		for _, elem := range x {
			redactAny(elem, paths, sensitivePaths, prefixes)
		}
	default:
		return
	}
}

// pathPrefixes lists every proper prefix of the sensitive paths, the paths worth descending into.
func pathPrefixes(sensitivePaths map[string]bool) map[string]bool {
	prefixes := make(map[string]bool)
	for path := range sensitivePaths {
		for i := range len(path) {
			if path[i] == '.' {
				prefixes[path[:i]] = true
			}
		}
	}
	return prefixes
}
