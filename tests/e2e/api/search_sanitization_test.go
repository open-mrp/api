//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchSanitization_FulltextEndpoints_DoNot500OnOperatorNoise(t *testing.T) {
	t.Parallel()

	paths := []string{
		rolesPath,
		transactionsPath,
		settlementsPath,
	}
	queries := []string{"+++", "---", "@@@", "((()))"}

	for _, path := range paths {
		for _, q := range queries {
			path := path
			q := q

			t.Run(path+"_q="+q, func(t *testing.T) {
				t.Parallel()

				status, body, err := apiClient.GetListRaw(path, url.Values{"q": {q}})
				require.NoError(t, err)
				requireStatus(t, 200, status, body)
			})
		}
	}
}

func TestSearchSanitization_LikeEndpoints_DoNot500OnLikeMetacharacters(t *testing.T) {
	t.Parallel()

	queries := []string{"%", "_", `\`, "%_%", "50%_off"}
	for _, q := range queries {
		q := q

		t.Run("q="+q, func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"q": {q}})
			require.NoError(t, err)
			requireStatus(t, 200, status, body)
		})
	}
}

func TestSearchSanitization_ListQueryTooLong_Returns400(t *testing.T) {
	t.Parallel()

	tooLongQuery := strings.Repeat("a", 501)
	status, body, err := apiClient.GetListRaw(itemsPath, url.Values{"q": {tooLongQuery}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}
