//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const agentMemoriesPath = "/v1/ai/memories"

func createMemory(t *testing.T, body map[string]any) string {
	t.Helper()
	status, respBody, err := apiClient.Post(agentMemoriesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	id := jsonField(parseJSON(respBody), "id")
	require.NotEmpty(t, id, "created memory should have an id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(agentMemoriesPath + "/" + id) })
	return id
}

func TestAgentMemories_ImportanceRejectsOutOfRangeAndNonTenthIncrements(t *testing.T) {
	t.Parallel()

	invalid := []float64{-0.1, 1.1, 1.5, 0.05, 0.15, 0.333, 0.25}
	for _, importance := range invalid {
		status, respBody, err := apiClient.Post(agentMemoriesPath, map[string]any{
			"category":   "preference",
			"content":    "Customer prefers express shipping.",
			"importance": importance,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, respBody)
	}
}

func TestAgentMemories_ImportanceAcceptsTenthIncrementsInRange(t *testing.T) {
	t.Parallel()

	valid := []float64{0, 0.1, 0.3, 0.5, 0.8, 1.0}
	for _, importance := range valid {
		createMemory(t, map[string]any{
			"category":   "fact",
			"content":    "Customer is on net-30 terms.",
			"importance": importance,
		})
	}
}

func TestAgentMemories_ImportanceOptionalOnCreate(t *testing.T) {
	t.Parallel()

	createMemory(t, map[string]any{
		"category": "instruction",
		"content":  "Always confirm the shipping address before quoting freight.",
	})
}

func TestAgentMemories_UpdateImportanceValidated(t *testing.T) {
	t.Parallel()

	id := createMemory(t, map[string]any{
		"category":   "preference",
		"content":    "Customer prefers email over phone.",
		"importance": 0.5,
	})

	status, respBody, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{"importance": 0.05}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	status, respBody, err = apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{"importance": 0.9}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
}

func TestAgentMemories_CategoryRestrictedToKnownValues(t *testing.T) {
	t.Parallel()

	status, respBody, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category": "insight",
		"content":  "This is not a recognized category.",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)

	for _, category := range []string{"preference", "fact", "instruction"} {
		createMemory(t, map[string]any{
			"category": category,
			"content":  "A memory in a recognized category.",
		})
	}
}
