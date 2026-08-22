package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

// serviceLevelsCanonicalPath is the dedicated paginated endpoint that the `service_levels` include's next_page_url points at when the inline list is truncated. Keeping this in one place so the URL shape stays consistent with the registered HTTP route.
const serviceLevelsCanonicalPath = "/v1/operations/carriers"

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeCarrier,
		Load:       resourceloaders.LoadCarriers,
		Subs: []resourcekit.SubField{
			// owner: no fetch. Builds the Owner shell from the carrier's account_id in LoadMeta (set by the carrier loader). type=system for NULL, type=account for a populated id with a stub Account.
			{
				Key:      "owner",
				Populate: populateOwnerOnCarrier,
			},
			// owner.account: fetches the full Account by id and writes it into the shell built by the "owner" sub above. The resolver implicitly fires "owner" first because the tree treats "owner.account" as having "owner" as an ancestor.
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromCarrier,
				Populate:    populateOwnerAccountOnCarrier,
			},
			// service_levels: fetches the previewed service levels via the SL loader, wraps them in *List[ServiceLevel] with a next_page URL pointing at the dedicated SL paginated endpoint when more exist beyond the preview cap.
			{
				Key:         "service_levels",
				Target:      constants.ObjectTypeServiceLevel,
				Cardinality: resourcekit.CardinalityList,
				ExtractIDs:  extractServiceLevelIDsFromCarrier,
				Populate:    populateServiceLevelsOnCarrier,
			},
		},
	})
}

func populateOwnerOnCarrier(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Carrier)
	accountID, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeCarrier, c.ID, "owner_account_id")
	c.Owner = buildOwnerShell(accountID)
}

func extractOwnerAccountIDFromCarrier(ctx context.Context, parent any) []string {
	c := parent.(*apiresource.Carrier)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeCarrier, c.ID, "owner_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOwnerAccountOnCarrier(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.Carrier)
	// The Owner shell is built by the "owner" sub which runs first (it's the dot-path ancestor of "owner.account" so tree.Has("owner") is true whenever tree.Has("owner.account") is). It has no Account attached — we read the FK from LoadMeta and look it up in `loaded`.
	if c.Owner == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeCarrier, c.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		c.Owner.Account = v.(*apiresource.Account)
	}
}

func extractServiceLevelIDsFromCarrier(ctx context.Context, parent any) []string {
	c := parent.(*apiresource.Carrier)
	ids, _ := resourcekit.GetLoadMeta(ctx).
		GetStrings(constants.ObjectTypeCarrier, c.ID, "service_level_ids")
	return ids
}

func populateServiceLevelsOnCarrier(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.Carrier)
	meta := resourcekit.GetLoadMeta(ctx)
	ids, _ := meta.GetStrings(constants.ObjectTypeCarrier, c.ID, "service_level_ids")
	hasMore, _ := meta.GetBool(constants.ObjectTypeCarrier, c.ID, "service_levels_has_more")

	items := make([]apiresource.ServiceLevel, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.ServiceLevel)))
		}
	}

	pageInfo := apiresource.PageInfo{HasNextPage: hasMore}
	if hasMore {
		// The inline preview isn't cursor-based, just a head-N projection of the full SL list. next_page_url therefore points at the first page of the dedicated paginated endpoint (no cursor query param); clients walk forward from there using its own cursors.
		nextURL := serviceLevelsCanonicalPath + "/" + c.ID + "/service-levels"
		pageInfo.NextPageURL = &nextURL
	}
	c.ServiceLevels = apiresource.NewList(items, pageInfo)
}
