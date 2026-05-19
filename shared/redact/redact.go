package redact

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
)

// SensitiveFields collects dot-separated JSON field paths declared with sensitive:"true"
// on structs reachable from root type typ. Root may be a pointer (e.g. *MyRequest); non-struct
// roots return nil.
//
// Embedding without a JSON key name preserves the same path prefix so promoted fields align
// with encoding/json flattened output.
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
		isSensitive := sf.Tag.Get("sensitive") == "true"
		ftd := deref(ft)

		if isSensitive {
			out[path] = true
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
		}
	}
}

// RedactJSON replaces JSON values whose paths exactly match sensitivePaths keys with the
// JSON string ****. Arrays reuse the parent's path segment so structs under an array resolve
// the same dotted paths encoding/json emits (no index in the path).
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

	redactAny(root, "", sensitivePaths)

	out, err := json.Marshal(root)
	if err != nil {
		return nil
	}
	return out
}

func redactAny(v any, path string, sensitivePaths map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			cur := pathJoin(path, k)
			if sensitivePaths[cur] {
				x[k] = "****"
				continue
			}
			redactAny(child, cur, sensitivePaths)
		}
	case []any:
		for _, elem := range x {
			redactAny(elem, path, sensitivePaths)
		}
	default:
		return
	}
}
