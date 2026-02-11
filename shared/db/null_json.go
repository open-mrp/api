package db

import (
	"database/sql/driver"
	"fmt"
)

type NullableRawMessage []byte

func (n *NullableRawMessage) Scan(value interface{}) error {
	if value == nil {
		*n = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into NullableRawMessage", value)
	}
	*n = NullableRawMessage(b)
	return nil
}

func (n NullableRawMessage) Value() (driver.Value, error) {
	if n == nil {
		return nil, nil
	}
	return []byte(n), nil
}
