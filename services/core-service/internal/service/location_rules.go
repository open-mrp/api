package service

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// locationShape is what a location will be once a create or update lands, checked as a whole.
type locationShape struct {
	// id is empty on create.
	id        string
	typeCode  string
	status    constants.ArchiveStatus
	parentID  *string
	addressID *string
	childIDs  []string
	// newParent is set when the write assigns parentID rather than keeping the current one.
	newParent bool
	// wasDefaultSite is set when the location is the account's default site.
	wasDefaultSite bool
}

// checkLocationShape enforces the site rules and the tree's integrity for a location about to be written:
//
//  1. A site is top-level, and only a site has an address.
//  2. The account's default site stays an active site.
//  3. A newly assigned parent is one of the account's active locations and not the location or one of its
//     descendants; children are the account's locations, never a site, the location itself or its ancestors.
//  4. The address is one of the account's.
func checkLocationShape(ctx context.Context, repos domain.RepoFactory, accountID string, s locationShape) *apierror.APIError {
	isSite := s.typeCode == string(constants.LocationTypeCodeSite)
	if isSite && s.parentID != nil {
		return apierror.NewFieldError("parent_id", apierror.CodeInvalidFormat,
			"A site is a top-level location, so it can't have a parent. Send `parent_id: null`.")
	}
	if !isSite && s.addressID != nil {
		return apierror.NewFieldError("address_id", apierror.CodeInvalidFormat,
			"Only a site has an address. Send `address_id: null`, or set `type` to `site`.")
	}
	if s.wasDefaultSite && !isSite {
		return apierror.NewFieldError("type", apierror.CodeInvalidFormat,
			"This location is your default site, so its type must stay `site`.")
	}
	if s.wasDefaultSite && !s.status.IsActive() {
		return apierror.NewFieldError("status", apierror.CodeInvalidFormat,
			"This location is your default site, so it can't be archived.")
	}

	repo := repos.NewLocationRepo()
	if s.parentID != nil && s.newParent {
		if *s.parentID == s.id {
			return apierror.NewFieldError("parent_id", apierror.CodeInvalidFormat, "A location cannot be its own parent.")
		}
		parents, apiErr := repo.GetByIDs(ctx, accountID, []string{*s.parentID})
		if apiErr != nil {
			return apiErr
		}
		if len(parents) == 0 {
			return apierror.NewFieldError("parent_id", apierror.CodeInvalidFormat, "Parent location not found.")
		}
		if apiErr := domain.RequireAssignable("parent_id", "location", parents[0].Status); apiErr != nil {
			return apiErr
		}
	}

	var ancestors map[string]bool
	if s.parentID != nil && (s.newParent || len(s.childIDs) > 0) {
		nodes, apiErr := repo.ListTreeNodes(ctx, accountID)
		if apiErr != nil {
			return apiErr
		}
		ancestors = locationAncestors(nodes, s.parentID)
		if s.id != "" && ancestors[s.id] {
			return apierror.NewFieldError("parent_id", apierror.CodeInvalidFormat,
				"A location can't be placed under one of its own descendants.")
		}
	}

	if len(s.childIDs) > 0 {
		children, apiErr := repo.GetByIDs(ctx, accountID, s.childIDs)
		if apiErr != nil {
			return apiErr
		}
		found := make(map[string]*domain.Location, len(children))
		for _, c := range children {
			found[c.ID] = c
		}
		for _, childID := range s.childIDs {
			child, ok := found[childID]
			switch {
			case !ok:
				return apierror.NewFieldError("child_ids", apierror.CodeInvalidFormat, "Child location not found.")
			case child.IsSite():
				return apierror.NewFieldError("child_ids", apierror.CodeInvalidFormat,
					"A site is a top-level location, so it can't be placed under another location.")
			case childID == s.id || ancestors[childID] || (s.parentID != nil && childID == *s.parentID):
				return apierror.NewFieldError("child_ids", apierror.CodeInvalidFormat,
					"A location can't be placed under itself or one of its own descendants.")
			}
		}
	}

	if s.addressID != nil {
		ok, apiErr := repos.NewAddressRepo().IsInAccount(ctx, accountID, *s.addressID)
		if apiErr != nil {
			return apiErr
		}
		if !ok {
			return apierror.NewFieldError("address_id", apierror.CodeInvalidFormat, "Address not found.")
		}
	}
	return nil
}

// locationAncestors is parentID and every location above it.
func locationAncestors(nodes []domain.LocationTreeNode, parentID *string) map[string]bool {
	byID := make(map[string]domain.LocationTreeNode, len(nodes))
	for _, n := range nodes {
		byID[n.ID] = n
	}
	out := map[string]bool{}
	for id := parentID; id != nil && !out[*id]; {
		out[*id] = true
		node, ok := byID[*id]
		if !ok {
			break
		}
		id = node.ParentID
	}
	return out
}

// claimDefaultSite makes a site the account's default when the account has none yet.
func claimDefaultSite(ctx context.Context, repos domain.RepoFactory, accountID string, loc *domain.Location) *apierror.APIError {
	if !loc.IsSite() || !loc.Status.IsActive() {
		return nil
	}
	_, apiErr := repos.NewLocationRepo().SetDefaultSiteIfUnset(ctx, accountID, loc.ID)
	return apiErr
}

// isDefaultSite reports whether id is the account's default site.
func isDefaultSite(ctx context.Context, repos domain.RepoFactory, accountID, id string) (bool, *apierror.APIError) {
	def, apiErr := repos.NewLocationRepo().DefaultSiteID(ctx, accountID)
	if apiErr != nil {
		return false, apiErr
	}
	return def != nil && *def == id, nil
}
