//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Gap coverage for notification preferences: the global default (null/omitted
// category via Clearable) and a category-specific override coexisting, plus
// upsert stability on repeat.

func TestNotificationPreferences_GlobalDefaultAndCategoryCoexist(t *testing.T) {
	t.Parallel()
	user := loginAsUser(t, seedUser2Email, seedUserPassword, SeedAccountID)

	// Global default: category omitted.
	global := upsertPreference(t, user, map[string]any{
		"in_app_enabled": true,
		"email_enabled":  false,
		"digest":         "off",
	})
	assert.Equal(t, "notification_preference", jsonField(global, "object"))
	assertNilField(t, global, "category")
	globalID := jsonField(global, "id")

	// Category-specific override.
	scoped := upsertPreference(t, user, map[string]any{
		"category":       "order.updated",
		"in_app_enabled": false,
		"email_enabled":  true,
		"digest":         "daily",
	})
	assert.Equal(t, "order.updated", jsonField(scoped, "category"))
	assert.NotEqual(t, globalID, jsonField(scoped, "id"), "the global default and a category override are distinct rows")

	// Re-upserting the same category is stable (same id, updated values).
	reupsert := upsertPreference(t, user, map[string]any{
		"category":       "order.updated",
		"in_app_enabled": true,
		"email_enabled":  true,
		"digest":         "instant",
	})
	assert.Equal(t, jsonField(scoped, "id"), jsonField(reupsert, "id"), "re-upserting a category replaces the same row")
	assert.Equal(t, "instant", jsonField(reupsert, "digest"))

	// Both surface in the list.
	list, _, err := user.GetList(preferencesPath, nil)
	require.NoError(t, err)
	var sawGlobal, sawScoped bool
	for _, raw := range list.Data {
		switch jsonField(parseJSON(raw), "id") {
		case globalID:
			sawGlobal = true
		case jsonField(scoped, "id"):
			sawScoped = true
		}
	}
	assert.True(t, sawGlobal, "the global default appears in the preference list")
	assert.True(t, sawScoped, "the category override appears in the preference list")
}
