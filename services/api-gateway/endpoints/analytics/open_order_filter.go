package analyticsep

import pb "github.com/open-mrp/api/shared/proto/core"

// OpenOrderFilters are the filters every open-orders report accepts. Every list is empty-means-all and they combine with AND.
type OpenOrderFilters struct {
	// Only count orders from these customers. Their child accounts are included.
	CustomerIDs []string `json:"customer_ids,omitzero"`
	// Only count orders from customers in these groups.
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// Only count orders owned by these sales reps (account users). A sales rep calling always sees only their own orders.
	SalesRepIDs []string `json:"sales_rep_ids,omitzero"`
	// Only count lines in these product lines. An order with no such line is left out.
	ProductLineIDs []string `json:"product_line_ids,omitzero"`
	// Only count lines for these items. An order with no such line is left out.
	ItemIDs []string `json:"item_ids,omitzero"`
}

func (f OpenOrderFilters) toProto() *pb.OpenOrderFilterProto {
	return &pb.OpenOrderFilterProto{
		CustomerIds:      f.CustomerIDs,
		CustomerGroupIds: f.CustomerGroupIDs,
		SalesRepIds:      f.SalesRepIDs,
		ProductLineIds:   f.ProductLineIDs,
		ItemIds:          f.ItemIDs,
	}
}
