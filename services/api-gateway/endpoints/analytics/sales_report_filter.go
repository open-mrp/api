package analyticsep

import (
	"time"

	"github.com/open-mrp/api/shared/field"
)

// SalesReportFilters are the entity filters every sales report accepts. Every list is empty-means-all and they combine with AND.
type SalesReportFilters struct {
	// Only count sales to these customers. Their child accounts are included.
	CustomerIDs []string `json:"customer_ids,omitzero"`
	// Only count sales to customers in these groups.
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// Only count lines in these product lines.
	ProductLineIDs []string `json:"product_line_ids,omitzero"`
	// Only count orders owned by these sales reps (account users). A sales rep calling always sees only their own sales.
	SalesRepIDs []string `json:"sales_rep_ids,omitzero"`
	// Only count lines for these items.
	ItemIDs []string `json:"item_ids,omitzero"`
}

// SalesComparisonPeriod is an optional second period reported beside the first. Set both bounds or neither.
type SalesComparisonPeriod struct {
	// Start of the comparison period, inclusive.
	ComparisonStartDate field.Optional[time.Time] `json:"comparison_starts_at,omitzero"`
	// End of the comparison period, inclusive.
	ComparisonEndDate field.Optional[time.Time] `json:"comparison_ends_at,omitzero"`
}
