package field

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
)

func newTestValidator(t *testing.T) *validator.Validate {
	t.Helper()
	v := validator.New()
	RegisterValidator(v)
	return v
}

type validatorReq struct {
	Email  Clearable[string]    `json:"email,omitzero" validate:"omitempty,max=5"`
	Tags   Clearable[[]string]  `json:"tags,omitzero" validate:"omitempty,max=2"`
	Days   Clearable[int32]     `json:"days,omitzero" validate:"omitempty,gte=0,lte=10"`
	Amount Clearable[float64]   `json:"amount,omitzero" validate:"omitempty,gt=0"`
	Name   Optional[string]     `json:"name,omitzero" validate:"omitempty,max=5"`
	Count  Optional[int]        `json:"count,omitzero" validate:"omitempty,lte=3"`
	Total  Optional[int64]      `json:"total,omitzero" validate:"omitempty,lte=3"`
	Ratio  Optional[float64]    `json:"ratio,omitzero" validate:"omitempty,gt=0"`
	Slots  Optional[int32]      `json:"slots,omitzero" validate:"omitempty,gte=1"`
	Active Optional[bool]       `json:"active,omitzero" validate:"omitempty,eq=true"`
	At     Optional[time.Time]  `json:"at,omitzero" validate:"omitempty"`
	Until  Clearable[time.Time] `json:"until,omitzero" validate:"omitempty"`
}

// TestRegisterValidator_setValueEnforcesInnerTags pins that a set wrapper hands the INNER
// value to the comparison tags. Every registered scalar type is exercised: an unregistered
// one panics ("Bad field type field.Optional[T]") the moment a comparison tag reaches it.
func TestRegisterValidator_setValueEnforcesInnerTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     validatorReq
		wantErr bool
	}{
		{"clearable string over max", validatorReq{Email: Set(strings.Repeat("x", 6))}, true},
		{"clearable string within max", validatorReq{Email: Set("ok")}, false},
		{"clearable slice over max", validatorReq{Tags: Set([]string{"a", "b", "c"})}, true},
		{"clearable slice within max", validatorReq{Tags: Set([]string{"a", "b"})}, false},
		{"clearable int32 under gte", validatorReq{Days: Set(int32(-1))}, true},
		{"clearable int32 over lte", validatorReq{Days: Set(int32(11))}, true},
		{"clearable int32 in range", validatorReq{Days: Set(int32(10))}, false},
		{"clearable float64 not gt", validatorReq{Amount: Set(-1.5)}, true},
		{"clearable float64 gt", validatorReq{Amount: Set(0.5)}, false},
		{"optional string over max", validatorReq{Name: Some(strings.Repeat("x", 6))}, true},
		{"optional int over lte", validatorReq{Count: Some(4)}, true},
		{"optional int within lte", validatorReq{Count: Some(3)}, false},
		{"optional int64 over lte", validatorReq{Total: Some(int64(4))}, true},
		{"optional float64 not gt", validatorReq{Ratio: Some(-2.5)}, true},
		{"optional int32 under gte", validatorReq{Slots: Some(int32(-1))}, true},
		{"optional bool eq", validatorReq{Active: Some(true)}, false},
		// A set-false is the inner type's zero value, so omitempty short-circuits eq=true.
		// The case is here for the registration itself: an unregistered bool panics on eq.
		{"optional bool set false", validatorReq{Active: Some(false)}, false},
		{"optional time set", validatorReq{At: Some(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC))}, false},
		{"clearable time set", validatorReq{Until: Set(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC))}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := newTestValidator(t).Struct(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

// TestRegisterValidator_unsetAndClearAreEmpty pins that both non-set states read as empty for
// omitempty, so a PATCH that omits a field never trips its bounds.
func TestRegisterValidator_unsetAndClearAreEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  validatorReq
	}{
		{"all unset", validatorReq{}},
		// A cleared int32 would fail gte=1 if the zero value leaked through as the inner value.
		{"cleared fields", validatorReq{
			Email:  Clear[string](),
			Tags:   Clear[[]string](),
			Days:   Clear[int32](),
			Amount: Clear[float64](),
			Until:  Clear[time.Time](),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := newTestValidator(t).Struct(&tt.req); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

type requiredReq struct {
	Email Clearable[string] `json:"email,omitzero" validate:"required"`
	Name  Optional[string]  `json:"name,omitzero" validate:"required"`
}

func TestRegisterValidator_required(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     requiredReq
		wantErr bool
	}{
		{"both unset", requiredReq{}, true},
		{"clearable unset", requiredReq{Name: Some("n")}, true},
		{"clearable cleared", requiredReq{Email: Clear[string](), Name: Some("n")}, true},
		{"optional unset", requiredReq{Email: Set("e")}, true},
		// A set-but-blank value is the inner value's own emptiness, so required still fires.
		{"set blank", requiredReq{Email: Set(""), Name: Some("n")}, true},
		{"both set", requiredReq{Email: Set("e"), Name: Some("n")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := newTestValidator(t).Struct(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

// registeredInnerTypes mirrors RegisterValidator's list. Keys omit the "field." qualifier so
// this table is not itself picked up by the repo scan below.
var registeredInnerTypes = map[string]bool{
	"Clearable[string]":    true,
	"Clearable[[]string]":  true,
	"Clearable[int]":       true,
	"Clearable[int32]":     true,
	"Clearable[int64]":     true,
	"Clearable[float64]":   true,
	"Clearable[bool]":      true,
	"Clearable[time.Time]": true,
	"Optional[string]":     true,
	"Optional[[]string]":   true,
	"Optional[int]":        true,
	"Optional[int32]":      true,
	"Optional[int64]":      true,
	"Optional[float64]":    true,
	"Optional[bool]":       true,
	"Optional[time.Time]":  true,
}

var (
	comparisonTagRE = regexp.MustCompile(`validate:"[^"]*\b(?:eq|ne|gt|gte|lt|lte|min|max|len)=`)
	wrapperFieldRE  = regexp.MustCompile(`field\.(Clearable|Optional)\[((?:\[\])?[\w.]+)\]`)
)

// TestRegisterValidator_coversRepoComparisonTags fails when a struct anywhere in the module
// puts a comparison tag on a wrapper whose inner type RegisterValidator does not register.
// The validator then sees the wrapper struct instead of the inner value and panics on the
// first request that reaches the endpoint, which no compile or unit test would catch.
func TestRegisterValidator_coversRepoComparisonTags(t *testing.T) {
	t.Parallel()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("module root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Skipf("module root not readable: %v", err)
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".pb.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(src), "\n") {
			if !comparisonTagRE.MatchString(line) {
				continue
			}
			for _, m := range wrapperFieldRE.FindAllStringSubmatch(line, -1) {
				if inner := m[1] + "[" + m[2] + "]"; !registeredInnerTypes[inner] {
					t.Errorf("%s:%d: field.%s carries a comparison tag but is not registered in RegisterValidator (validation panics at request time); register it there and in registeredInnerTypes", path, i+1, inner)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
