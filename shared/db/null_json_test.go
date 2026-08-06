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
	n := NullableRawMessage(`{"stale":true}`)
	if err := n.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if n != nil {
		t.Fatalf("Scan(nil) = %q, want nil", string(n))
	}
}
