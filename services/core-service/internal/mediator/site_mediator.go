package mediator

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
)

var siteMedTracer = tracing.GetTracer("core-service.site_mediator")

type siteMedImpl struct {
	repos domain.RepoFactory
}

type SiteMedConfig struct {
	// Repos (required) is the repository factory locations and the account's default site are read through.
	Repos domain.RepoFactory
}

func (c *SiteMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("site mediator: repos is required")
	}
	return nil
}

func NewSiteMed(config *SiteMedConfig) domain.SiteMed {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &siteMedImpl{repos: config.Repos}
}

// DefaultSiteID is the account's default site, nil while it has none.
func (m *siteMedImpl) DefaultSiteID(ctx context.Context, accountID string) (*string, *apierror.APIError) {
	ctx, span := siteMedTracer.Start(ctx, "mediator.site.default_site_id")
	defer span.End()

	id, apiErr := m.repos.NewLocationRepo().DefaultSiteID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return id, nil
}

// Resolve is the site a document is assigned: siteID when the caller names one, else the account's default site.
//
//  1. With a siteID, return a validation error on param unless it is one of the account's locations and a site.
//  2. Return a 422 resource_archived on param when that site is archived.
//  3. Without one, return the account's default site (nil while it has none).
func (m *siteMedImpl) Resolve(ctx context.Context, accountID string, siteID *string, param string) (*string, *apierror.APIError) {
	ctx, span := siteMedTracer.Start(ctx, "mediator.site.resolve")
	defer span.End()

	if siteID == nil {
		return m.DefaultSiteID(ctx, accountID)
	}
	locations, apiErr := m.repos.NewLocationRepo().GetByIDs(ctx, accountID, []string{*siteID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(locations) == 0 {
		return nil, tracing.Trace(span, apierror.NewFieldError(param, apierror.CodeInvalidFormat, "Site not found."))
	}
	site := locations[0]
	if !site.IsSite() {
		return nil, tracing.Trace(span, apierror.NewFieldError(param, apierror.CodeInvalidFormat,
			"The location is not a site. Choose a location whose `type` is `site`."))
	}
	if apiErr := domain.RequireAssignable(param, "site", site.Status); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &site.ID, nil
}

// LocationIDsInSites is every location of the account that belongs to one of siteIDs, the sites included.
//
//  1. Load the account's location tree and default site.
//  2. Keep each location whose nearest site ancestor is in siteIDs; one under no site belongs to the default site.
func (m *siteMedImpl) LocationIDsInSites(ctx context.Context, accountID string, siteIDs []string) ([]string, *apierror.APIError) {
	ctx, span := siteMedTracer.Start(ctx, "mediator.site.location_ids_in_sites")
	defer span.End()

	if len(siteIDs) == 0 {
		return nil, nil
	}
	repo := m.repos.NewLocationRepo()
	nodes, apiErr := repo.ListTreeNodes(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defaultSiteID, apiErr := repo.DefaultSiteID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var fallback string
	if defaultSiteID != nil {
		fallback = *defaultSiteID
	}

	byID := make(map[string]domain.LocationTreeNode, len(nodes))
	for _, n := range nodes {
		byID[n.ID] = n
	}
	wanted := make(map[string]bool, len(siteIDs))
	for _, id := range siteIDs {
		wanted[id] = true
	}
	var out []string
	for _, n := range nodes {
		if site := domain.SiteOf(byID, n.ID, fallback); site != "" && wanted[site] {
			out = append(out, n.ID)
		}
	}
	return out, nil
}
