package domain

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/constants"
)

func TestLegacyMovementType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code    string
		measure string
		want    constants.InventoryMovementType
	}{
		{"scan", "-3", constants.InventoryMovementTypeConsumption},
		{"scan", "5", constants.InventoryMovementTypeOutput},
		{"system_action", "10", constants.InventoryMovementTypeReceipt},
		{"user_action", "0", constants.InventoryMovementTypeAdjustment},
		{"user_correction", "-1", constants.InventoryMovementTypeAdjustment},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, LegacyMovementType(tc.code, decimal.RequireFromString(tc.measure)), tc.code+" "+tc.measure)
	}
}

func TestLegacyMovementActor(t *testing.T) {
	t.Parallel()
	station, user, key, agent, unknown := "scst_1", "us_1", "apke_1", "agdf_1", "zz_1"

	a := LegacyMovementActor(&station, &user)
	assert.Equal(t, constants.ActorTypeDevice, a.Type, "a scan is recorded by its station")
	assert.Equal(t, station, a.ID)

	assert.Equal(t, constants.ActorTypeUser, LegacyMovementActor(nil, &user).Type)
	assert.Equal(t, constants.ActorTypeAPIKey, LegacyMovementActor(nil, &key).Type)
	assert.Equal(t, constants.ActorTypeAgent, LegacyMovementActor(nil, &agent).Type)
	assert.Equal(t, constants.ActorTypeSystem, LegacyMovementActor(nil, &unknown).Type)
	assert.Equal(t, constants.ActorTypeSystem, LegacyMovementActor(nil, nil).Type)
}
