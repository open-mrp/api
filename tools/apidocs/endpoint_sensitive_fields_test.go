package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/redact"
)

// TestEndpointSensitiveFieldsAreRedactable walks the request and response type of every registered endpoint and fails when a sensitive:"true" field does not resolve to a path redact.SensitiveFields reports.
//
// The redactor is what keeps secrets out of the request log, and it fails silently: a sensitive field placed somewhere its path scheme cannot express (a map value, a wrapper type that stores its value unexported, an untyped payload) produces no path, no error, and a logged plaintext body.
func TestEndpointSensitiveFieldsAreRedactable(t *testing.T) {
	t.Parallel()
	groups := buildAllGroups()

	found := 0
	for _, group := range groups {
		for _, ep := range group.Endpoints {
			for _, typ := range []reflect.Type{ep.GetRequestType(), ep.GetResponseType()} {
				if typ == nil {
					continue
				}
				paths := redact.SensitiveFields(typ)
				for _, site := range sensitiveSites(typ) {
					found++
					where := fmt.Sprintf("%s %s (%s): %s", ep.GetMethod(), ep.GetRoute(), typ, site.field)
					switch {
					case site.reason != "":
						t.Errorf("%s is tagged sensitive but %s, so redact.SensitiveFields cannot produce a path for it and the value would be logged in full", where, site.reason)
					case !paths[site.path]:
						t.Errorf("%s is tagged sensitive but redact.SensitiveFields did not report its JSON path %q (reported: %v)", where, site.path, paths)
					}
				}
			}
		}
	}

	// A walk that stops finding the fields it is meant to police would pass silently.
	if found == 0 {
		t.Fatal("no sensitive fields found across the endpoint surface; the walk is no longer reaching request and response types")
	}
}

// sensitiveSite is one sensitive:"true" field found by an independent walk of a type. reason is empty when the field sits where the redactor's dotted-path scheme can address it.
type sensitiveSite struct {
	path   string
	field  string
	reason string
}

const maxWalkDepth = 32

func sensitiveSites(typ reflect.Type) []sensitiveSite {
	var out []sensitiveSite
	walkSensitiveSites(typ, "", "", make(map[reflect.Type]bool), &out, 0)
	return out
}

func walkSensitiveSites(typ reflect.Type, prefix, reason string, visited map[reflect.Type]bool, out *[]sensitiveSite, depth int) {
	typ = derefType(typ)
	if typ == nil || typ.Kind() != reflect.Struct || depth > maxWalkDepth || visited[typ] {
		return
	}
	visited[typ] = true
	defer delete(visited, typ)

	for i := range typ.NumField() {
		sf := typ.Field(i)
		if !sf.IsExported() {
			continue
		}

		name, skip := jsonFieldName(sf)
		if sf.Anonymous {
			if skip {
				continue
			}
			embeddedPrefix := prefix
			if name != "" {
				embeddedPrefix = joinPath(prefix, name)
			}
			walkSensitiveSites(sf.Type, embeddedPrefix, reason, visited, out, depth+1)
			continue
		}
		if skip || name == "" {
			continue
		}

		path := joinPath(prefix, name)
		field := fmt.Sprintf("%s.%s", typ, sf.Name)

		if tag, ok := sf.Tag.Lookup("sensitive"); ok {
			if tag != "true" {
				*out = append(*out, sensitiveSite{path: path, field: field, reason: fmt.Sprintf("its sensitive tag reads %q rather than \"true\"", tag)})
				continue
			}
			*out = append(*out, sensitiveSite{path: path, field: field, reason: reason})
			continue
		}

		ft := derefType(sf.Type)
		switch ft.Kind() {
		case reflect.Struct:
			if inner, wrapped := unexportedWrapperInner(ft); wrapped {
				walkSensitiveSites(inner, path, fmt.Sprintf("it is stored inside %s, whose value is unexported", ft), visited, out, depth+1)
				continue
			}
			walkSensitiveSites(ft, path, reason, visited, out, depth+1)
		case reflect.Slice, reflect.Array:
			walkSensitiveSites(ft.Elem(), path, reason, visited, out, depth+1)
		case reflect.Map:
			walkSensitiveSites(ft.Elem(), path, "it sits under a map key, which no dotted path can name", visited, out, depth+1)
		}
	}
}

// unexportedWrapperInner resolves the value type carried by wrappers such as field.Optional and field.Clearable, whose storage is unexported and therefore invisible to a reflective field walk.
func unexportedWrapperInner(typ reflect.Type) (reflect.Type, bool) {
	if typ.NumField() == 0 {
		return nil, false
	}
	for i := range typ.NumField() {
		if typ.Field(i).IsExported() {
			return nil, false
		}
	}
	resolver, ok := reflect.New(typ).Elem().Interface().(interface{ OpenAPIInnerType() reflect.Type })
	if !ok {
		return nil, false
	}
	inner := resolver.OpenAPIInnerType()
	if inner == nil {
		return nil, false
	}
	return inner, true
}

func derefType(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

func jsonFieldName(sf reflect.StructField) (name string, skip bool) {
	tag, ok := sf.Tag.Lookup("json")
	if !ok || tag == "" {
		return "", false
	}
	name, _, _ = strings.Cut(tag, ",")
	name = strings.TrimSpace(name)
	if name == "-" {
		return "", true
	}
	return name, false
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// TestSensitiveSites_detectsUnreachablePlacements guards the guard: the walk above only protects the endpoint surface if it still notices the placements redact cannot express.
func TestSensitiveSites_detectsUnreachablePlacements(t *testing.T) {
	t.Parallel()

	type leaf struct {
		Secret string `json:"secret" sensitive:"true"` // #nosec G101 — test fixture
	}
	type inMap struct {
		Creds map[string]leaf `json:"creds"`
	}
	type typoTag struct {
		Secret string `json:"secret" sensitive:"yes"` // #nosec G101 — test fixture
	}
	type inOptional struct {
		Config field.Optional[leaf] `json:"config,omitzero"`
	}
	type reachable struct {
		Items []leaf `json:"items"`
	}
	type taggedOptional struct {
		Password field.Optional[string] `json:"password,omitzero" sensitive:"true"` // #nosec G101 — test fixture
	}

	tests := []struct {
		name       string
		typ        reflect.Type
		wantPath   string
		wantReason bool
	}{
		{"map value", reflect.TypeFor[inMap](), "creds.secret", true},
		{"inside an optional wrapper", reflect.TypeFor[inOptional](), "config.secret", true},
		{"misspelled tag value", reflect.TypeFor[typoTag](), "secret", true},
		{"slice of structs", reflect.TypeFor[reachable](), "items.secret", false},
		{"optional scalar tagged directly", reflect.TypeFor[taggedOptional](), "password", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sites := sensitiveSites(tt.typ)
			if len(sites) != 1 {
				t.Fatalf("got %d sites, want 1: %+v", len(sites), sites)
			}
			if sites[0].path != tt.wantPath {
				t.Errorf("path = %q, want %q", sites[0].path, tt.wantPath)
			}
			if (sites[0].reason != "") != tt.wantReason {
				t.Errorf("reason = %q, wantReason = %v", sites[0].reason, tt.wantReason)
			}
		})
	}

	// json.RawMessage and any are opaque to reflection; a secret inside one can only be masked by tagging the whole field, which the walk reports as reachable.
	type opaque struct {
		Input json.RawMessage `json:"input" sensitive:"true"`
	}
	sites := sensitiveSites(reflect.TypeFor[opaque]())
	if len(sites) != 1 || sites[0].reason != "" || sites[0].path != "input" {
		t.Fatalf("got %+v, want a single reachable site at input", sites)
	}
}
