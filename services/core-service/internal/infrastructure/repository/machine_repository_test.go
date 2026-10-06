package repository

import (
	"testing"

	"github.com/open-mrp/api/shared/pagination"
)

func TestMachBuildSearchParams_MatchesANamePrefixLiterally(t *testing.T) {
	t.Parallel()

	if got := machBuildSearchParams(new("")); got.Valid {
		t.Errorf("an empty term searched for %q", got.String)
	}
	if got, want := machBuildSearchParams(new(`1_%`)), `1\_\%%`; !got.Valid || got.String != want {
		t.Errorf("pattern = %q, want %q", got.String, want)
	}
}

// A cursor issued without a tier comes from an unsearched list, where every machine is tier 0; a NULL tier
// would compare false against every row and end the list.
func TestMachCursorMatchTier_DefaultsToTierZero(t *testing.T) {
	t.Parallel()

	if got := machCursorMatchTier(pagination.StringCursor{}); !got.Valid || got.Int64 != 0 {
		t.Errorf("untiered cursor = %+v, want tier 0", got)
	}
	if got := machCursorMatchTier(pagination.StringCursor{MatchTier: new(1)}); !got.Valid || got.Int64 != 1 {
		t.Errorf("tiered cursor = %+v, want tier 1", got)
	}
}
