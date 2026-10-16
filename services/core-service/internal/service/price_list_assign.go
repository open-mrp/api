package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// checkAssignableLists refuses assigning lists that aren't the account's (404 on param) or are archived (422
// resource_archived). Lists already assigned (held) may stay even when archived.
func checkAssignableLists(ctx context.Context, repos domain.RepoFactory, accountID, param string, ids, held []string) *apierror.APIError {
	if len(ids) > domain.MaxPriceListAssignments {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, fmt.Sprintf("At most %d price lists can be assigned.", domain.MaxPriceListAssignments))
	}
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return nil
	}
	lists, apiErr := repos.NewPriceListRepo().GetByIDs(ctx, accountID, ids)
	if apiErr != nil {
		return apiErr
	}
	byID := make(map[string]*domain.PriceList, len(lists))
	for _, l := range lists {
		byID[l.ID] = l
	}
	heldSet := make(map[string]bool, len(held))
	for _, h := range held {
		heldSet[h] = true
	}
	for _, listID := range ids {
		list := byID[listID]
		if list == nil {
			return apierror.NewNotFoundError("No price list found with the provided ID.").WithParam(param)
		}
		if heldSet[listID] {
			continue
		}
		if apiErr := domain.RequireAssignable(param, "price list", list.Status); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// assignCustomerLists replaces the lists assigned to a customer directly.
func assignCustomerLists(ctx context.Context, repos domain.RepoFactory, accountID, relationID string, listIDs []string) *apierror.APIError {
	if apiErr := ensurePriceListsSynced(ctx, repos, accountID); apiErr != nil {
		return apiErr
	}
	return repos.NewPriceListRepo().SetCustomerLists(ctx, relationID, listIDs)
}

// assignGroupLists replaces the lists assigned to an account group.
func assignGroupLists(ctx context.Context, repos domain.RepoFactory, accountID, groupID string, listIDs []string) *apierror.APIError {
	if apiErr := ensurePriceListsSynced(ctx, repos, accountID); apiErr != nil {
		return apiErr
	}
	return repos.NewPriceListRepo().SetGroupLists(ctx, groupID, listIDs)
}

// dropCustomerPriceLists deletes deleted or merged-away customers' own lists (their mirrored account prices) and
// their list assignments.
func dropCustomerPriceLists(ctx context.Context, repos domain.RepoFactory, accountID string, relationIDs []string) *apierror.APIError {
	repo := repos.NewPriceListRepo()
	for _, rel := range relationIDs {
		list, apiErr := repo.GetBySource(ctx, accountID, constants.PriceListSourceCustomer, rel)
		if apiErr != nil {
			return apiErr
		}
		if list != nil {
			if apiErr := repo.Delete(ctx, accountID, list.ID); apiErr != nil {
				return apiErr
			}
		}
	}
	return repo.DeleteCustomerAssignments(ctx, relationIDs)
}
