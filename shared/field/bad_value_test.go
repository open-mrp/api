package field

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

type badValueLine struct {
	Quantity Optional[int32] `json:"quantity,omitzero"`
	Note     string          `json:"note"`
}

type badValueEmbedded struct {
	Ref Optional[string] `json:"ref,omitzero"`
}

type badValueReq struct {
	badValueEmbedded
	Count    Optional[int32]            `json:"count,omitzero"`
	Name     Optional[string]           `json:"name,omitzero"`
	Due      Optional[time.Time]        `json:"due,omitzero"`
	Cleared  Clearable[time.Time]       `json:"cleared,omitzero"`
	Plain    int32                      `json:"plain"`
	Line     Optional[badValueLine]     `json:"line,omitzero"`
	Lines    []badValueLine             `json:"lines"`
	Patch    Clearable[badValueLine]    `json:"patch,omitzero"`
	Tags     Optional[[]string]         `json:"tags,omitzero"`
	Limits   map[string]Optional[int32] `json:"limits"`
	Skipped  Optional[int32]            `json:"-"`
	Untagged Optional[int32]            // encoding/json keys it by the Go name
}

// The decoder reports an error raised inside UnmarshalJSON with no field, so the path has to come
// from the body. These are the shapes a request body actually takes.
func TestBadValuePath(t *testing.T) {
	t.Parallel()
	int32Type := reflect.TypeFor[int32]()

	tests := []struct {
		name     string
		body     string
		wantPath string
		wantType reflect.Type // the UnmarshalTypeError's Go type; nil when the cause is another error
	}{
		{"optional int given a string", `{"count":"abc"}`, "count", int32Type},
		{"optional int given a fraction", `{"count":1.5}`, "count", int32Type},
		{"optional int out of range", `{"count":2147483648}`, "count", int32Type},
		{"optional string given a number", `{"name":12}`, "name", reflect.TypeFor[string]()},
		{"optional time given a number", `{"due":12}`, "due", reflect.TypeFor[time.Time]()},
		{"plain field", `{"plain":"x"}`, "plain", int32Type},
		{"field of an embedded struct", `{"ref":1}`, "ref", reflect.TypeFor[string]()},
		{"field inside an optional object", `{"line":{"quantity":"x"}}`, "line.quantity", int32Type},
		{"optional object given a scalar", `{"line":"x"}`, "line", reflect.TypeFor[badValueLine]()},
		{"field of a list item", `{"lines":[{"quantity":1},{"quantity":"x"}]}`, "lines[1].quantity", int32Type},
		{"plain field of a list item", `{"lines":[{"note":1}]}`, "lines[0].note", reflect.TypeFor[string]()},
		{"field inside a clearable object", `{"patch":{"quantity":true}}`, "patch.quantity", int32Type},
		{"item of an optional list", `{"tags":["a",2]}`, "tags[1]", reflect.TypeFor[string]()},
		{"value of a map", `{"limits":{"b":1,"a":"x"}}`, "limits.a", int32Type},
		{"untagged field", `{"Untagged":"x"}`, "Untagged", int32Type},
		{"key matched ignoring case", `{"COUNT":"x"}`, "count", int32Type},
		{"first bad field in declaration order", `{"name":1,"count":"x"}`, "count", int32Type},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := json.Unmarshal([]byte(tt.body), &badValueReq{}); err == nil {
				t.Fatalf("precondition: %s must fail to decode", tt.body)
			}
			path, err := BadValuePath([]byte(tt.body), &badValueReq{})
			if path != tt.wantPath {
				t.Fatalf("path = %q, want %q (err %v)", path, tt.wantPath, err)
			}
			uterr, ok := errors.AsType[*json.UnmarshalTypeError](err)
			if !ok {
				t.Fatalf("cause = %T %v, want *json.UnmarshalTypeError", err, err)
			}
			if uterr.Type != tt.wantType {
				t.Fatalf("cause type = %v, want %v", uterr.Type, tt.wantType)
			}
		})
	}
}

func TestBadValuePath_timeParseErrors(t *testing.T) {
	t.Parallel()
	for body, want := range map[string]string{
		`{"due":"yesterday"}`:                  "due",
		`{"cleared":"2024-13-45"}`:             "cleared",
		`{"line":{"quantity":1},"due":"soon"}`: "due",
	} {
		t.Run(want, func(t *testing.T) {
			t.Parallel()
			path, err := BadValuePath([]byte(body), &badValueReq{})
			if path != want {
				t.Fatalf("path = %q, want %q", path, want)
			}
			if _, ok := errors.AsType[*time.ParseError](err); !ok {
				t.Fatalf("cause = %T %v, want *time.ParseError", err, err)
			}
		})
	}
}

func TestBadValuePath_explicitNull(t *testing.T) {
	t.Parallel()
	path, err := BadValuePath([]byte(`{"line":{"quantity":null}}`), &badValueReq{})
	if path != "line.quantity" || !errors.Is(err, ErrExplicitNull) {
		t.Fatalf("got (%q, %v), want (line.quantity, ErrExplicitNull)", path, err)
	}
}

// When no single field is at fault the caller keeps the decoder's own error.
func TestBadValuePath_noFieldAtFault(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"valid body":         `{"count":1,"cleared":null,"lines":[{"quantity":2}]}`,
		"body is an array":   `[]`,
		"body is a string":   `"x"`,
		"key json ignores":   `{"-":"x"}`,
		"unknown key":        `{"bogus":"x"}`,
		"malformed json":     `{"count":`,
		"empty body":         ``,
		"clearable set null": `{"patch":null}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if path, err := BadValuePath([]byte(body), &badValueReq{}); path != "" || err != nil {
				t.Fatalf("got (%q, %v), want no field", path, err)
			}
		})
	}
}
