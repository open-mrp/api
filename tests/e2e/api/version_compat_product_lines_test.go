//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// preview.6 made a product line's commission_policy nullable, since it is withheld from customer and supplier portal users. A portal pinned to preview.5 reads "" in its place; the seller's own callers read the real policy on both versions.

func TestVersionCompat_ProductLines_PortalReadsWithheldCommissionAsAnEmptyString(t *testing.T) {
	t.Parallel()
	path := productLinesPath + "/" + SeedProductLineID

	for who, portal := range portalClients(t) {
		got := parseJSON(mustGetAs(t, portal.WithAPIVersion(preview5APIVersion), path, nil))
		assert.Equal(t, "", got["commission_policy"], who)
		assert.Nil(t, got["notes"], "%s: notes were already nullable and stay null", who)
	}
	for _, apiVersion := range []string{preview5APIVersion, defaultAPIVersion} {
		got := parseJSON(mustGetAs(t, apiClient.WithAPIVersion(apiVersion), path, nil))
		assert.Contains(t, []string{"commission_applied", "commission_exempt"}, got["commission_policy"], apiVersion)
	}
}
