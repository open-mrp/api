package redact_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/redact"
)

// sentinel is planted in every sensitive field of a fixture; surviving anywhere in the redacted body is the leak these tests exist to catch.
const sentinel = "SENTINEL-SECRET-VALUE"

type optionalScalarReq struct {
	Email    string                 `json:"email"`
	Password field.Optional[string] `json:"password,omitzero" sensitive:"true"` // #nosec G101 — test fixture
}

type taggedMapReq struct {
	ID      string            `json:"id"`
	Headers map[string]string `json:"headers" sensitive:"true"`
}

type outerSlice struct {
	Groups []sliceReq `json:"groups"`
}

type deepReq struct {
	Level1 struct {
		Level2 struct {
			Inner inner `json:"inner"`
		} `json:"level2"`
	} `json:"level1"`
}

// emittedPaths returns every dot-separated path encoding/json produced for raw, collapsing array elements onto the parent path the way RedactJSON matches them.
func emittedPaths(t *testing.T, raw []byte) map[string]bool {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		t.Fatalf("decode marshaled fixture: %v", err)
	}
	out := make(map[string]bool)
	var walk func(v any, prefix string)
	walk = func(v any, prefix string) {
		switch x := v.(type) {
		case map[string]any:
			for k, child := range x {
				path := k
				if prefix != "" {
					path = prefix + "." + k
				}
				out[path] = true
				walk(child, path)
			}
		case []any:
			for _, elem := range x {
				walk(elem, prefix)
			}
		}
	}
	walk(root, "")
	return out
}

// assertSensitiveRoundTrip checks both directions of the type-to-path mapping: every path SensitiveFields reports is a path encoding/json really emits, and every sensitive value reachable in the fixture is masked by RedactJSON.
func assertSensitiveRoundTrip(t *testing.T, v any) {
	t.Helper()

	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if !strings.Contains(string(raw), sentinel) {
		t.Fatalf("fixture does not marshal the sentinel, nothing is being proven: %s", raw)
	}

	paths := redact.SensitiveFields(reflect.TypeOf(v))
	if len(paths) == 0 {
		t.Fatalf("SensitiveFields returned no paths for %T", v)
	}

	emitted := emittedPaths(t, raw)
	for p := range paths {
		if !emitted[p] {
			t.Errorf("SensitiveFields reported path %q, which encoding/json never emits for %T (emitted: %v)", p, v, emitted)
		}
	}

	out := redact.RedactJSON(raw, paths)
	if out == nil {
		t.Fatalf("RedactJSON returned nil for %s", raw)
	}
	if strings.Contains(string(out), sentinel) {
		t.Errorf("sensitive value survived redaction of %T: %s", v, out)
	}
}

func TestSensitiveFields_roundTripsWithJSONOutput(t *testing.T) {
	t.Parallel()

	secret := sentinel
	var deep deepReq
	deep.Level1.Level2.Inner = inner{Token: sentinel, Name: "n"}

	tests := []struct {
		name  string
		value any
	}{
		{"flat", flatReq{User: "u", Password: sentinel}},
		{"pointer root", &flatReq{User: "u", Password: sentinel}},
		{"nested struct", nestedReq{ID: "x", Inner: inner{Token: sentinel, Name: "n"}}},
		{"pointer field", ptrField{Value: &secret}},
		{"slice of structs", sliceReq{Items: []item{{ID: "1", Secret: sentinel}, {ID: "2", Secret: sentinel}}}},
		{"nested slices", outerSlice{Groups: []sliceReq{{Items: []item{{ID: "1", Secret: sentinel}}}}}},
		{"multiple paths", twoSens{Login: "l", Password: sentinel, OTP: sentinel}},
		{"embedded promoted", embedPromoted{EmbedInnerSensitive: EmbedInnerSensitive{S: sentinel}, Key: "k"}},
		{"embedded named", embedNamed{EmbedInnerSensitive: EmbedInnerSensitive{S: sentinel}, Key: "k"}},
		{"optional scalar", optionalScalarReq{Email: "e", Password: field.Some(sentinel)}},
		{"whole map tagged", taggedMapReq{ID: "x", Headers: map[string]string{"authorization": sentinel}}},
		{"deeply nested struct", deep},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertSensitiveRoundTrip(t, tt.value)
		})
	}
}

func TestSensitiveFields_emptySliceStillYieldsPath(t *testing.T) {
	t.Parallel()
	// The path comes from the type, so an empty collection at runtime must not change it — otherwise a first request with no items would train nothing and a later populated one would leak.
	m := redact.SensitiveFields(reflect.TypeFor[sliceReq]())
	if len(m) != 1 || !m["items.secret"] {
		t.Fatalf("got %#v want items.secret", m)
	}
	out := redact.RedactJSON([]byte(`{"items":[]}`), m)
	if string(out) != `{"items":[]}` {
		t.Fatalf("got %s", out)
	}
}

func TestSensitiveFields_unexportedSensitiveFieldIgnored(t *testing.T) {
	t.Parallel()
	type withUnexported struct {
		Public string `json:"public"`
		secret string `sensitive:"true"` // #nosec G101 — test fixture
	}
	// The tagged unexported field must be populated somewhere or it reads as dead weight; it exists to prove SensitiveFields skips what json can never marshal.
	_ = withUnexported{Public: "p", secret: "s"}
	if m := redact.SensitiveFields(reflect.TypeFor[withUnexported]()); m != nil {
		t.Fatalf("unexported fields are never marshaled; want nil, got %#v", m)
	}
}

func TestRedactJSON_maskedValueIsAlwaysAString(t *testing.T) {
	t.Parallel()
	// A numeric or object secret must come back as the "****" string, never a partially-masked value of its original type.
	out := redact.RedactJSON(
		[]byte(`{"pin":1234,"creds":{"user":"u","pass":"p"},"list":[1,2]}`),
		map[string]bool{"pin": true, "creds": true, "list": true},
	)
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"pin", "creds", "list"} {
		if got[k] != "****" {
			t.Errorf("%s: got %#v want ****", k, got[k])
		}
	}
}

func TestRedactJSON_nullAndMissingSensitiveKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"null value is masked", `{"password":null}`, `{"password":"****"}`},
		{"absent key adds nothing", `{"user":"u"}`, `{"user":"u"}`},
		{"empty object", `{}`, `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := redact.RedactJSON([]byte(tt.in), map[string]bool{"password": true})
			if string(out) != tt.want {
				t.Fatalf("got %s want %s", out, tt.want)
			}
		})
	}
}
