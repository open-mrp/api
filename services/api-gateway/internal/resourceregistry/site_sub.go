package resourceregistry

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

// SiteSub is the `site` include of a document that happens at one site (sales orders, purchase orders, receiving,
// picks, shipments, jobs, work orders): the site location stashed by resourceloaders.StashSiteID. idOf gives the
// document's ID and setSite writes the loaded site onto it. Register the document's `site` include key against
// constants.ObjectTypeLocation alongside it.
func SiteSub(parentType constants.ObjectType, idOf func(parent any) string, setSite func(parent any, site *apiresource.Location)) include.SubField {
	siteID := func(ctx context.Context, parent any) string {
		id, _ := include.GetLoadMeta(ctx).GetString(parentType, idOf(parent), resourceloaders.SiteMetaKey)
		return id
	}
	return include.SubField{
		Key:         "site",
		Target:      constants.ObjectTypeLocation,
		Cardinality: include.CardinalityOnePtr,
		ExtractIDs: func(ctx context.Context, parent any) []string {
			if id := siteID(ctx, parent); id != "" {
				return []string{id}
			}
			return nil
		},
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			if v, ok := loaded[siteID(ctx, parent)]; ok {
				setSite(parent, v.(*apiresource.Location))
			}
		},
	}
}
