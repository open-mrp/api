//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// preview.6 made a conversation's legal_hold nullable, since it is withheld from customer and supplier portal users. A portal pinned to preview.5 reads "" in its place; the seller's own users read the real status on both versions.

// Not parallel: the support route these cases open through is account-wide.
func TestVersionCompat_Conversations_PortalReadsWithheldLegalHoldAsAnEmptyString(t *testing.T) {
	seedSupportRoute(t)
	staff := chatUserClient(t)

	for who, portal := range portalClients(t) {
		convID := openSupportCase(t, portal)
		path := conversationsPath + "/" + convID

		got := parseJSON(mustGetAs(t, portal.WithAPIVersion(preview5APIVersion), path, nil))
		assert.Equal(t, "", got["legal_hold"], who)

		for _, apiVersion := range []string{preview5APIVersion, defaultAPIVersion} {
			got = parseJSON(mustGetAs(t, staff.WithAPIVersion(apiVersion), path, nil))
			assert.Equal(t, "released", got["legal_hold"], "%s: the seller reads the case's legal hold", apiVersion)
		}
	}
}
