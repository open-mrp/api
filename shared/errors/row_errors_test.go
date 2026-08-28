package apierror

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// Bulk services collect failures into a `var rowErrs apierror.RowErrors`, so the zero value
// has to be usable without a constructor.
func TestRowErrors_ZeroValue(t *testing.T) {
	t.Parallel()
	var rowErrs RowErrors

	if rowErrs.Any() {
		t.Error("Any() = true on a zero value, want false")
	}
	if entries := rowErrs.Entries(); entries != nil {
		t.Errorf("Entries() = %v, want nil", entries)
	}
	if summary := rowErrs.Summary("products"); summary != nil {
		t.Errorf("Summary() = %v, want nil", summary)
	}

	rowErrs.AddValidation(3, "rows[3].sku", "SKU is required.")

	if !rowErrs.Any() {
		t.Error("Any() = false after a failure was recorded, want true")
	}
	if got := len(rowErrs.Entries()); got != 1 {
		t.Fatalf("len(Entries()) = %d, want 1", got)
	}
	if got := rowErrs.Entries()[0].Index; got != 3 {
		t.Errorf("Index = %d, want 3", got)
	}
	if summary := rowErrs.Summary("products"); summary == nil {
		t.Error("Summary() = nil after a failure was recorded, want an error")
	}
}

func TestRowErrors_AddValidation(t *testing.T) {
	t.Parallel()
	var rowErrs RowErrors
	rowErrs.AddValidation(0, "rows[0].name", "Name is required.")

	entry := rowErrs.Entries()[0]
	if entry.Error.Code != ErrorCodeValidationFailed {
		t.Errorf("Code = %q, want %q", entry.Error.Code, ErrorCodeValidationFailed)
	}
	if entry.Error.Type != ErrorTypeInvalidRequest {
		t.Errorf("Type = %q, want %q", entry.Error.Type, ErrorTypeInvalidRequest)
	}
	if entry.Error.Message != "Name is required." {
		t.Errorf("Message = %q, want %q", entry.Error.Message, "Name is required.")
	}
	if entry.Error.Param == nil || *entry.Error.Param != "rows[0].name" {
		t.Errorf("Param = %v, want %q", entry.Error.Param, "rows[0].name")
	}
	if entry.Error.DocURL == nil || *entry.Error.DocURL != docURLValidationFailed {
		t.Errorf("DocURL = %v, want %q", entry.Error.DocURL, docURLValidationFailed)
	}
	if entry.Error.IsTransient {
		t.Error("IsTransient = true, want false")
	}
}

// A row failure recorded from a nil error must not panic; it degrades to the zero error object.
func TestNewRowError_NilAPIError(t *testing.T) {
	t.Parallel()
	rowErr := NewRowError(2, nil)

	if rowErr.Index != 2 {
		t.Errorf("Index = %d, want 2", rowErr.Index)
	}
	if rowErr.Error != (ResponseError{}) {
		t.Errorf("Error = %+v, want the zero value", rowErr.Error)
	}
}

func TestRowErrors_Summary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		add          func(*RowErrors)
		wantNil      bool
		wantMessage  string
		wantParam    string
		wantNilParam bool
	}{
		{
			name:    "no failures yields no error",
			add:     func(*RowErrors) {},
			wantNil: true,
		},
		{
			name: "single failure names its param",
			add: func(e *RowErrors) {
				e.AddValidation(0, "rows[0].sku", "SKU is required.")
			},
			wantMessage: "Invalid products — rows[0].sku: SKU is required..",
			wantParam:   "rows[0].sku",
		},
		{
			name: "clauses keep the order the rows were recorded in",
			add: func(e *RowErrors) {
				e.AddValidation(2, "rows[2].sku", "SKU is required.")
				e.AddValidation(0, "rows[0].name", "Name is required.")
				e.AddValidation(1, "rows[1].qty", "Quantity must be positive.")
			},
			wantMessage: "Invalid products — rows[2].sku: SKU is required.; rows[0].name: Name is required.; rows[1].qty: Quantity must be positive..",
			wantParam:   "rows[2].sku",
		},
		{
			name: "a failure with no param contributes only its message",
			add: func(e *RowErrors) {
				e.Add(0, NewResourceNotFoundError("Product not found."))
				e.AddValidation(1, "rows[1].sku", "SKU is required.")
			},
			wantMessage: "Invalid products — Product not found.; rows[1].sku: SKU is required..",
			// The first row carries no param, so the first row that does names the error.
			wantParam: "rows[1].sku",
		},
		{
			name: "no failure carries a param",
			add: func(e *RowErrors) {
				e.Add(0, NewResourceNotFoundError("Product not found."))
				e.Add(1, NewResourceConflictError("Product already exists."))
			},
			wantMessage:  "Invalid products — Product not found.; Product already exists..",
			wantNilParam: true,
		},
		{
			name: "an empty message still occupies its clause",
			add: func(e *RowErrors) {
				e.AddValidation(0, "rows[0].sku", "")
				e.AddValidation(1, "rows[1].sku", "SKU is required.")
			},
			wantMessage: "Invalid products — rows[0].sku: ; rows[1].sku: SKU is required..",
			wantParam:   "rows[0].sku",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rowErrs RowErrors
			tt.add(&rowErrs)

			summary := rowErrs.Summary("products")
			if tt.wantNil {
				if summary != nil {
					t.Fatalf("Summary() = %v, want nil", summary)
				}
				return
			}
			if summary == nil {
				t.Fatal("Summary() = nil, want an error")
			}

			if summary.PublicMessage != tt.wantMessage {
				t.Errorf("PublicMessage = %q, want %q", summary.PublicMessage, tt.wantMessage)
			}
			if summary.Param != tt.wantParam {
				t.Errorf("Param = %q, want %q", summary.Param, tt.wantParam)
			}
			if summary.Code != ErrorCodeValidationFailed {
				t.Errorf("Code = %q, want %q", summary.Code, ErrorCodeValidationFailed)
			}
			if summary.Type != ErrorTypeInvalidRequest {
				t.Errorf("Type = %q, want %q", summary.Type, ErrorTypeInvalidRequest)
			}
			if got := GetHTTPStatusCode(summary.Code); got != 400 {
				t.Errorf("status = %d, want 400", got)
			}
			if summary.Stack != "" {
				t.Error("expected no stack on a 4xx summary")
			}

			resp := summary.ToResponseError()
			if tt.wantNilParam {
				if resp.Param != nil {
					t.Errorf("response Param = %q, want null", *resp.Param)
				}
			} else if resp.Param == nil || *resp.Param != tt.wantParam {
				t.Errorf("response Param = %v, want %q", resp.Param, tt.wantParam)
			}
		})
	}
}

// The summary must stay whole at bulk scale: every row's clause is present, in order, with
// no truncation or elision.
func TestRowErrors_Summary_AtBulkCap(t *testing.T) {
	t.Parallel()
	const rows = 1000

	var rowErrs RowErrors
	for i := range rows {
		rowErrs.AddValidation(i, fmt.Sprintf("rows[%d].sku", i), "SKU is required.")
	}

	if got := len(rowErrs.Entries()); got != rows {
		t.Fatalf("len(Entries()) = %d, want %d", got, rows)
	}

	summary := rowErrs.Summary("products")
	if summary == nil {
		t.Fatal("Summary() = nil, want an error")
	}
	if summary.Param != "rows[0].sku" {
		t.Errorf("Param = %q, want %q", summary.Param, "rows[0].sku")
	}
	if got := strings.Count(summary.PublicMessage, "; "); got != rows-1 {
		t.Errorf("clause separator count = %d, want %d", got, rows-1)
	}
	if !strings.HasPrefix(summary.PublicMessage, "Invalid products — rows[0].sku: SKU is required.; ") {
		t.Errorf("message does not open with the first row's clause: %q", summary.PublicMessage[:80])
	}
	if !strings.HasSuffix(summary.PublicMessage, "; rows[999].sku: SKU is required..") {
		t.Errorf("message does not close with the last row's clause: %q", summary.PublicMessage[len(summary.PublicMessage)-80:])
	}
	for _, i := range []int{0, 1, 499, 998, 999} {
		clause := "rows[" + strconv.Itoa(i) + "].sku: SKU is required."
		if !strings.Contains(summary.PublicMessage, clause) {
			t.Errorf("message is missing the clause for row %d", i)
		}
	}
}
