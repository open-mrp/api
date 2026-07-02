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
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into NullableRawMessage", value)
	}
	// The database/sql layer hands custom Scanners the driver's own buffer, which some drivers (notably go-sql-driver/mysql) reuse for the next read on the same pooled connection. Retaining it lets a concurrent query overwrite these bytes mid-flight, so the value must be copied to stay valid past this call.
	cp := make([]byte, len(b))
	copy(cp, b)
	*n = NullableRawMessage(cp)
	return nil
}

func (n NullableRawMessage) Value() (driver.Value, error) {
	if n == nil {
		return nil, nil
	}
	return []byte(n), nil
}
