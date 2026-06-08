package field

import (
	"testing"
)

type OptionalReq struct {
	Name  Optional[string] `json:"name"`
	Phone Optional[string] `json:"phone"`
}

type embeddedOptionalReq struct {
	OptionalReq
	Other Clearable[string] `json:"other,omitzero"`
}

func TestExplicitNullField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		body     string
		v        any
		wantName string
		wantOK   bool
	}{
		{"explicit null", `{"phone": null}`, &OptionalReq{}, "phone", true},
		{"first null of many", `{"name": null, "phone": null}`, &OptionalReq{}, "name", true},
		{"value not null", `{"phone": "555"}`, &OptionalReq{}, "", false},
		{"absent key", `{}`, &OptionalReq{}, "", false},
		{"embedded field null", `{"name": null}`, &embeddedOptionalReq{}, "name", true},
		{"not an object", `[]`, &OptionalReq{}, "", false},
		{"non-nullable null ignored", `{"other": null}`, &embeddedOptionalReq{}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			name, ok := ExplicitNullField([]byte(tt.body), tt.v)
			if ok != tt.wantOK || name != tt.wantName {
				t.Fatalf("got (%q, %v), want (%q, %v)", name, ok, tt.wantName, tt.wantOK)
			}
		})
	}
}
