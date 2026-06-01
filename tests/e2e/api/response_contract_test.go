//go:build e2e

package api_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListEndpoints_ResponseContracts(t *testing.T) {
	t.Parallel()

	publicSpec, err := LoadPublicSpec()
	require.NoError(t, err)

	for _, ep := range listEndpoints {
		if shouldSkipPrivatePresenterContract(publicSpec, ep.Path) {
			continue
		}

		ep := ep
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()

			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			status, body, err := apiClient.GetListRaw(path, nil)
			require.NoError(t, err, "GET %s failed", path)
			skipOnNonClientError(t, path, status)
			if status != 200 {
				t.Skipf("GET %s returned %d", path, status)
				return
			}

			AssertResponseBodyValid(t, body)
		})
	}
}

func TestSeededGetEndpoints_ResponseContracts(t *testing.T) {
	t.Parallel()

	publicSpec, err := LoadPublicSpec()
	require.NoError(t, err)

	for prefix, seedID := range pathSpecificIDSeeds {
		if shouldSkipPrivatePresenterContract(publicSpec, seededGetPrefixToTemplate(prefix)) {
			continue
		}

		prefix := prefix
		seedID := seedID

		t.Run(strings.Trim(prefix, "/"), func(t *testing.T) {
			t.Parallel()

			path, ok := resolveSeededGetPath(prefix, seedID)
			if !ok {
				t.Skipf("Could not resolve seeded GET path for %s", prefix)
				return
			}

			status, body, err := apiClient.GetListRaw(path, nil)
			require.NoError(t, err, "GET %s failed", path)
			skipOnNonClientError(t, path, status)
			if status == 405 {
				t.Skipf("GET %s not supported", path)
				return
			}
			requireStatus(t, 200, status, body)

			AssertResponseBodyValid(t, body)
		})
	}
}

func resolveSeededGetPath(prefix, seedID string) (string, bool) {
	path := prefix
	for param, val := range pathParamSeeds {
		path = strings.ReplaceAll(path, "{"+param+"}", val)
	}
	if strings.Contains(path, "{id}") {
		path = strings.ReplaceAll(path, "{id}", seedID)
	}
	if strings.Contains(path, "{") {
		return "", false
	}
	if strings.HasSuffix(path, "/") {
		path += seedID
	}
	return path, true
}

func shouldSkipPrivatePresenterContract(publicSpec *openAPISpec, path string) bool {
	return !strings.HasPrefix(path, "/v1/auth/") && !specHasGETPath(publicSpec, path)
}

func specHasGETPath(spec *openAPISpec, path string) bool {
	if spec == nil {
		return false
	}
	methods, ok := spec.Paths[path]
	if !ok {
		return false
	}
	_, ok = methods["get"]
	return ok
}

func seededGetPrefixToTemplate(prefix string) string {
	if strings.HasSuffix(prefix, "/") {
		return prefix + "{id}"
	}
	return prefix
}
