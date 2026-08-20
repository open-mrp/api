package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Pins the wire values: these are the `key` column of `account_plan_limit` rows, so a rename here
// silently stops matching real rows rather than failing to compile.
func TestAccountPlanLimitKeyValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "invoices_maximum", string(AccountPlanLimitInvoicesMaximum))
	assert.Equal(t, "batches_maximum", string(AccountPlanLimitBatchesMaximum))
	assert.Equal(t, "seats_maximum", string(AccountPlanLimitSeatsMaximum))
	assert.Equal(t, "sandboxes_maximum", string(AccountPlanLimitSandboxesMaximum))
}
