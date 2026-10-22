package repository

import (
	"strings"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
)

// MATCH fails with 1191 when no FULLTEXT key covers exactly its column.
func TestCatalogSearchIndexes_AreDeclaredInMigrations(t *testing.T) {
	t.Parallel()

	schema := migrationsText(t)
	for _, index := range []string{"item_sku_ngram_idx", "item_description_ngram_idx"} {
		if !strings.Contains(schema, index) {
			t.Errorf("%s is MATCHed by the catalog search but no migration creates it", index)
		}
	}
}

// A search whose matches were read up front pages them by primary key and never tests the terms again.
func TestNewItemListQuery_SearchMatchesReplaceTheTerms(t *testing.T) {
	t.Parallel()

	term := "abc"
	q := newItemListQuery(domain.ListItemsParams{AccountID: "ac_1", Query: &term}, nil, []string{"it_1", "it_2"})
	where := strings.Join(q.where, " AND ")
	if strings.Contains(where, "LIKE") || !strings.Contains(where, "i.id IN (?,?)") {
		t.Errorf("the matches should replace the LIKE terms: %s", where)
	}
	if got := q.indexes(); got[len(got)-1] != "PRIMARY" {
		t.Errorf("the matches should be read by primary key: %v", got)
	}

	p := newProductListQuery(domain.ListProductsFullParams{AccountID: "ac_1", Query: &term}, nil, []string{"it_1"})
	where = strings.Join(p.where, " AND ")
	if strings.Contains(where, "LIKE") || !strings.Contains(where, "i.id IN (?)") {
		t.Errorf("the matches should replace the LIKE terms: %s", where)
	}
	if got := p.itemIndexes(); got[len(got)-1] != "PRIMARY" {
		t.Errorf("the matches should be read by primary key: %v", got)
	}

	unmatched := newItemListQuery(domain.ListItemsParams{AccountID: "ac_1", Query: &term}, nil, nil)
	if !strings.Contains(strings.Join(unmatched.where, " AND "), "i.sku LIKE ?") {
		t.Errorf("a search without matches should test each item: %v", unmatched.where)
	}
}
