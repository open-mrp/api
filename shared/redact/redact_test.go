package redact_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/redact"
)

type flatReq struct {
	User     string `json:"user"`
	Password string `json:"password" sensitive:"true"`
}

type inner struct {
	Token string `json:"token" sensitive:"true"`
	Name  string `json:"name"`
}

type nestedReq struct {
	ID    string `json:"id"`
	Inner inner  `json:"inner"`
}

type item struct {
	Secret string `json:"secret" sensitive:"true"`
	ID     string `json:"id"`
}

type sliceReq struct {
	Items []item `json:"items"`
}

type ptrField struct {
	Value *string `json:"value" sensitive:"true"`
}

type EmbedInnerSensitive struct {
	S string `json:"s" sensitive:"true"`
}

type embedPromoted struct {
	EmbedInnerSensitive        // promoted as top-level keys
	Key                 string `json:"key"`
}

type embedNamed struct {
	EmbedInnerSensitive `json:"wrapped"`
	Key                 string `json:"key"`
}

func TestSensitiveFields_flat(t *testing.T) {
	m := redact.SensitiveFields(reflect.TypeFor[flatReq]())
	if len(m) != 1 || !m["password"] {
		t.Fatalf("got %#v want password", m)
	}
}

func TestSensitiveFields_nested(t *testing.T) {
	m := redact.SensitiveFields(reflect.TypeFor[nestedReq]())
	if len(m) != 1 || !m["inner.token"] {
		t.Fatalf("got %#v want inner.token", m)
	}
}

func TestSensitiveFields_sliceOfStructs(t *testing.T) {
	m := redact.SensitiveFields(reflect.TypeFor[sliceReq]())
	if len(m) != 1 || !m["items.secret"] {
		t.Fatalf("got %#v want items.secret", m)
	}
}

func TestSensitiveFields_pointerReceiverType(t *testing.T) {
	m := redact.SensitiveFields(reflect.TypeFor[*flatReq]())
	if len(m) != 1 || !m["password"] {
		t.Fatalf("got %#v want password", m)
	}
}

func TestSensitiveFields_pointerSensitiveField(t *testing.T) {
	m := redact.SensitiveFields(reflect.TypeFor[ptrField]())
	if len(m) != 1 || !m["value"] {
		t.Fatalf("got %#v want value", m)
	}
}

func TestSensitiveFields_embeddedPromoted(t *testing.T) {
	m := redact.SensitiveFields(reflect.TypeFor[embedPromoted]())
	if len(m) != 1 || !m["s"] {
		t.Fatalf("got %#v want s", m)
	}
}

func TestSensitiveFields_embeddedNamedJSON(t *testing.T) {
	m := redact.SensitiveFields(reflect.TypeFor[embedNamed]())
	if len(m) != 1 || !m["wrapped.s"] {
		t.Fatalf("got %#v want wrapped.s", m)
	}
}

func TestSensitiveFields_nonStructRoot(t *testing.T) {
	if m := redact.SensitiveFields(reflect.TypeFor[string]()); m != nil {
		t.Fatalf("expected nil map, got %#v", m)
	}
}

func TestRedactJSON_basic(t *testing.T) {
	in := []byte(`{"user":"a","password":"hunter2"}`)
	out := redact.RedactJSON(in, map[string]bool{"password": true})
	want := []byte(`{"password":"****","user":"a"}`)
	if string(out) != string(want) {
		t.Fatalf("got %s want %s", out, want)
	}
}

func TestRedactJSON_nested(t *testing.T) {
	in := []byte(`{"id":"x","inner":{"name":"n","token":"abc"}}`)
	out := redact.RedactJSON(in, map[string]bool{"inner.token": true})
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	inner := got["inner"].(map[string]any)
	if inner["name"] != "n" || inner["token"] != "****" {
		t.Fatalf("got %#v", got)
	}
}

func TestRedactJSON_slice(t *testing.T) {
	in := []byte(`{"items":[{"id":"1","secret":"x"},{"id":"2","secret":"y"}]}`)
	out := redact.RedactJSON(in, map[string]bool{"items.secret": true})
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	items := got["items"].([]any)
	for i, it := range items {
		im := it.(map[string]any)
		if im["secret"] != "****" {
			t.Fatalf("item %d: %#v", i, im)
		}
	}
}

func TestRedactJSON_nestedObjectMaskedAsWhole_whenSensitivePathIsParent(t *testing.T) {
	// If caller marks "inner" sensitive, entire subtree is replaced — struct tag leaf-only in practice.
	out := redact.RedactJSON([]byte(`{"inner":{"token":"keep"}}`), map[string]bool{"inner": true})
	want := []byte(`{"inner":"****"}`)
	if string(out) != string(want) {
		t.Fatalf("got %s want %s", out, want)
	}
}

func TestSensitiveFields_noSensitiveTags_returnsNilMap(t *testing.T) {
	type plain struct {
		A string `json:"a"`
		B int    `json:"b"`
	}
	if m := redact.SensitiveFields(reflect.TypeFor[plain]()); m != nil {
		t.Fatalf("want nil map, got %#v", m)
	}
}

type twoSens struct {
	Login    string `json:"login"`
	Password string `json:"password" sensitive:"true"`
	OTP      string `json:"otp" sensitive:"true"`
}

func TestSensitiveFields_multiplePaths(t *testing.T) {
	m := redact.SensitiveFields(reflect.TypeFor[twoSens]())
	if len(m) != 2 || !m["password"] || !m["otp"] {
		t.Fatalf("got %#v", m)
	}
}

type dashJSON struct {
	LocalOnly string `json:"-" sensitive:"true"` // #nosec G101 — test fixture
	PublicRef string `json:"ref"`
}

func TestSensitiveFields_dashJSONName_skipsTag(t *testing.T) {
	if m := redact.SensitiveFields(reflect.TypeFor[dashJSON]()); m != nil {
		t.Fatalf("expected nil (no JSON-emitted sensitive paths), got %#v", m)
	}
}

func TestRedactJSON_preservesNumericField(t *testing.T) {
	out := redact.RedactJSON([]byte(`{"secret":"hide","count":42}`), map[string]bool{"secret": true})
	if out == nil {
		t.Fatal("got nil")
	}
	var got map[string]any
	dec := json.NewDecoder(strings.NewReader(string(out)))
	dec.UseNumber()
	if err := dec.Decode(&got); err != nil {
		t.Fatal(err)
	}
	v, ok := got["count"].(json.Number)
	if !ok || v != "42" {
		t.Fatalf("expected json.Number 42, got %#v (%T)", got["count"], got["count"])
	}
	if got["secret"] != "****" {
		t.Fatalf("secret got %#v", got["secret"])
	}
}

func TestRedactJSON_multiplePaths_sameObject(t *testing.T) {
	out := redact.RedactJSON(
		[]byte(`{"a":"keep","otp":"4444","password":"zzz"}`),
		map[string]bool{"otp": true, "password": true},
	)
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got["a"] != "keep" || got["otp"] != "****" || got["password"] != "****" {
		t.Fatalf("got %#v", got)
	}
}

func TestRedactJSON_rootArray_objectsRedacted(t *testing.T) {
	out := redact.RedactJSON([]byte(`[{"id":"1","token":"one"},{"token":"two"}]`),
		map[string]bool{"token": true})
	if out == nil {
		t.Fatal("nil")
	}
	var arr []map[string]any
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	for i, o := range arr {
		if o["token"] != "****" {
			t.Fatalf("idx %d: %#v", i, o)
		}
	}
}

func nestedWrapperJSON(n int, inner string) string {
	if n <= 0 {
		return inner
	}
	return `{"wrapper":` + nestedWrapperJSON(n-1, inner) + `}`
}

func TestRedactJSON_deepNestedPath(t *testing.T) {
	const depth = 12
	path := strings.Repeat("wrapper.", depth)
	path += "password"

	inner := `{"password":"deep","keep":true}`
	payload := nestedWrapperJSON(depth, inner)

	out := redact.RedactJSON([]byte(payload), map[string]bool{path: true})
	if out == nil {
		t.Fatal("expected redacted JSON, got nil")
	}

	dec := json.NewDecoder(strings.NewReader(string(out)))
	dec.UseNumber()
	var cursor any
	if err := dec.Decode(&cursor); err != nil {
		t.Fatal(err)
	}

	walk := cursor.(map[string]any)
	for range depth {
		walk = walk["wrapper"].(map[string]any)
	}
	bottom := walk
	if bottom["password"] != "****" || bottom["keep"] != true {
		t.Fatalf("bottom %#v", bottom)
	}
}

func TestRedactJSON_decoderIgnoresTrailingBytesAfterFirstValue(t *testing.T) {
	// encoding/json.Decoder stops after one value; trailing bytes are not an error.
	out := redact.RedactJSON([]byte(`{"a":"hide","b":1} trailing garbage`), map[string]bool{"a": true})
	if out == nil {
		t.Fatal("expected output")
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got["a"] != "****" || got["b"] != float64(1) {
		t.Fatalf("got %#v", got)
	}
}

func TestSensitiveFields_reflectsTaggedCreatedAPIKeyShape(t *testing.T) {
	type createdAPIKey struct {
		Object       string `json:"object"`
		APIKeySecret string `json:"api_key_secret" sensitive:"true"` // #nosec G101 — test fixture
	}
	m := redact.SensitiveFields(reflect.TypeFor[createdAPIKey]())
	if len(m) != 1 || !m["api_key_secret"] {
		t.Fatalf("got %#v want api_key_secret only", m)
	}
}

func TestRedactJSON_malformed(t *testing.T) {
	out := redact.RedactJSON([]byte(`{"a":`), map[string]bool{"a": true})
	if out != nil {
		t.Fatalf("want nil got %s", out)
	}
}

func TestRedactJSON_emptyPaths(t *testing.T) {
	in := []byte(`{"a":1}`)
	out := redact.RedactJSON(in, nil)
	if string(out) != string(in) {
		t.Fatalf("got %s", out)
	}
}
