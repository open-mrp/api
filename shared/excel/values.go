package excel

import (
	"strings"
	"time"
)

// separates a child list inside one cell, matching what the importer splits on
const nameSeparator = "; "

// flattens a child collection into a single cell
func JoinNames(names []string) string {
	return strings.Join(names, nameSeparator)
}

// dates every export filename
const dateLayout = "01-02-2006"

// names an export's file from its resource slug, e.g. "unit_groups_export_07-30-2026.xlsx".
// One separator throughout, so the name survives a URL path without escaping.
func Filename(slug string, at time.Time) string {
	return slug + "_export_" + at.Format(dateLayout) + ".xlsx"
}

// dereferences an optional string into a cell value, since a blank cell and an
// absent value are the same thing to Excel
func Str(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// renders an optional date the way the rest of a sheet does; an unset date is blank
func Date(v *time.Time) string {
	if v == nil {
		return ""
	}
	return v.Format(dateLayout)
}
