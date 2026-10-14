package fieldpb

import (
	"testing"

	"github.com/open-mrp/apikit/field"
	"github.com/stretchr/testify/assert"
)

type freightSection struct {
	CarrierID field.Clearable[string]
	Policy    field.Optional[string]
}

// Sending one field of a section changes only that field; the rest keep their stored values.
func TestApply_MergesASectionFieldByField(t *testing.T) {
	t.Parallel()
	storedCarrier, storedPolicy := new("cr_1"), "billed"

	patch := freightSection{Policy: field.Some("free")}
	assert.Equal(t, storedCarrier, ApplyClearable(storedCarrier, patch.CarrierID), "a left-out field keeps its value")
	assert.Equal(t, "free", ApplyOptional(storedPolicy, patch.Policy))

	assert.Nil(t, ApplyClearable(storedCarrier, field.Clear[string]()), "null clears")
	assert.Equal(t, "cr_2", *ApplyClearable(storedCarrier, field.Set("cr_2")))
	assert.Equal(t, storedPolicy, ApplyOptional(storedPolicy, field.None[string]()))
}
