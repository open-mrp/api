//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A station's batch search matches a substring of the batch item's SKU, case-insensitively, with LIKE
// wildcards taken literally. The seeded batch at the seeded station is of the LKN item.
func TestBatchesByStation_SearchMatchesItemSKU(t *testing.T) {
	t.Parallel()

	path := scanningStationsPath + "/" + SeedScanningStationID + "/batches"
	search := func(q string) []string {
		return listIDs(t, path, url.Values{"q": {q}, "limit": {"100"}})
	}

	assert.Contains(t, search("LKN"), SeedBatchID, "the item's whole SKU")
	assert.Contains(t, search("kn"), SeedBatchID, "a substring, in any case")
	assert.NotContains(t, search(SeedItemSKU), SeedBatchID, "another item's SKU")
	assert.NotContains(t, search("%"), SeedBatchID, "a LIKE wildcard is literal")
	assert.Empty(t, search(uniqueName("e2e-no-sku")), "a term no SKU contains matches no batch")
}
