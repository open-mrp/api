package db

import (
	"encoding/json"
	"testing"
)

// database/sql hands a Scanner the driver's own buffer, which the driver may reuse for
// the next row. Scan must copy so a scanned value survives later activity on the same
// connection overwriting that buffer. Regression for non-deterministic "invalid
// character" JSON errors when a scanned job's results were clobbered before use.
func TestNullableRawMessage_ScanCopiesBuffer(t *testing.T) {
	t.Parallel()
	src := []byte(`{"production_runs":[{"production_run_id":"pr_x"}]}`)

	var n NullableRawMessage
	if err := n.Scan(src); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	// Simulate the driver reusing the buffer for the next read.
	for i := range src {
		src[i] = 'X'
	}

	if !json.Valid(n) {
		t.Fatalf("scanned value was aliased and got corrupted: %q", string(n))
	}
	if got, want := string(n), `{"production_runs":[{"production_run_id":"pr_x"}]}`; got != want {
		t.Fatalf("scanned value = %q, want %q", got, want)
	}
}

func TestNullableRawMessage_ScanNil(t *testing.T) {
	t.Parallel()
	n := NullableRawMessage(`{"stale":true}`)
	if err := n.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if n != nil {
		t.Fatalf("Scan(nil) = %q, want nil", string(n))
	}
}

func TestNullableRawMessage_ScanRejectsUnsupportedTypes(t *testing.T) {
	t.Parallel()
	var n NullableRawMessage
	if err := n.Scan(42); err == nil {
		t.Fatal("expected an error scanning an int into NullableRawMessage")
	}
}

func TestNullableRawMessage_Value(t *testing.T) {
	t.Parallel()

	var absent NullableRawMessage
	got, err := absent.Value()
	if err != nil {
		t.Fatalf("Value() on nil: %v", err)
	}
	if got != nil {
		t.Fatalf("Value() on nil = %#v, want nil so the column is written as NULL", got)
	}

	present := NullableRawMessage(`{"a":1}`)
	got, err = present.Value()
	if err != nil {
		t.Fatalf("Value(): %v", err)
	}
	b, ok := got.([]byte)
	if !ok {
		t.Fatalf("Value() = %T, want []byte", got)
	}
	if string(b) != `{"a":1}` {
		t.Fatalf("Value() = %q, want %q", string(b), `{"a":1}`)
	}
}

// Value hands the driver the same backing array, so a caller that reuses its buffer after the
// write would change what is being written.
func TestNullableRawMessage_RoundTrip(t *testing.T) {
	t.Parallel()
	original := NullableRawMessage(`{"production_runs":[]}`)

	v, err := original.Value()
	if err != nil {
		t.Fatalf("Value(): %v", err)
	}

	var scanned NullableRawMessage
	if err := scanned.Scan(v); err != nil {
		t.Fatalf("Scan(Value()): %v", err)
	}
	if string(scanned) != string(original) {
		t.Fatalf("round trip = %q, want %q", string(scanned), string(original))
	}
}
