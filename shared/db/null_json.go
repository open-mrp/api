package db

import (
	"database/sql/driver"
	"fmt"
)

type NullableRawMessage []byte

func (n *NullableRawMessage) Scan(value any) error {
	if value == nil {
		*n = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into NullableRawMessage", value)
	}
	// Copy, don't alias. database/sql hands a Scanner the driver's own buffer, which is
	// only valid until the next row read — the mysql driver reuses it on the next query.
	// Keeping the alias lets later DB activity on the same connection overwrite this
	// value before it is used (e.g. a job's results scanned, then clobbered by the audit
	// and outbox writes that follow, before the job is marshaled), which surfaces as
	// non-deterministic "invalid character" JSON errors.
	cp := make([]byte, len(b))
	copy(cp, b)
	*n = NullableRawMessage(cp)
	return nil
}

// Value is a string, not []byte: with interpolateParams the driver writes []byte as a _binary
// literal, which MySQL refuses to store in a JSON column.
func (n NullableRawMessage) Value() (driver.Value, error) {
	if n == nil {
		return nil, nil
	}
	return string(n), nil
}
