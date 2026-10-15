package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// requireAssignableLocation rejects a work center location that isn't the account's (422 invalid_format) or is
// archived (422 resource_archived).
func requireAssignableLocation(ctx context.Context, repos domain.RepoFactory, accountID, param, locationID string) *apierror.APIError {
	locations, apiErr := repos.NewLocationRepo().GetByIDs(ctx, accountID, []string{locationID})
	if apiErr != nil {
		return apiErr
	}
	if len(locations) == 0 {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, "Location not found.")
	}
	return domain.RequireAssignable(param, "location", locations[0].Status)
}

// requireAssignableMachines rejects machine IDs that aren't the account's (422 invalid_format) or are archived (422
// resource_archived), so a work center never takes in an unknown or retired machine.
func requireAssignableMachines(ctx context.Context, repos domain.RepoFactory, accountID, param string, ids []string) *apierror.APIError {
	if len(ids) == 0 {
		return nil
	}
	machines, apiErr := repos.NewMachineRepo().GetByIDs(ctx, accountID, ids)
	if apiErr != nil {
		return apiErr
	}
	byID := make(map[string]*domain.Machine, len(machines))
	for _, m := range machines {
		byID[m.ID] = m
	}
	for _, id := range ids {
		m, ok := byID[id]
		if !ok {
			return apierror.NewFieldError(param, apierror.CodeInvalidFormat, fmt.Sprintf("Machine %q not found.", id))
		}
		if apiErr := domain.RequireAssignable(param, "machine", m.Status); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// requireKnownScanningStations rejects scanning station IDs that aren't the account's (422 invalid_format).
func requireKnownScanningStations(ctx context.Context, repos domain.RepoFactory, accountID, param string, ids []string) *apierror.APIError {
	if len(ids) == 0 {
		return nil
	}
	stations, apiErr := repos.NewScanningStationRepo().GetByIDs(ctx, accountID, ids)
	if apiErr != nil {
		return apiErr
	}
	found := make(map[string]bool, len(stations))
	for _, s := range stations {
		found[s.ID] = true
	}
	for _, id := range ids {
		if !found[id] {
			return apierror.NewFieldError(param, apierror.CodeInvalidFormat, fmt.Sprintf("Scanning station %q not found.", id))
		}
	}
	return nil
}

// machineStatus is a new machine's status: active unless the caller archived it.
func machineStatus(s *constants.ArchiveStatus) constants.ArchiveStatus {
	if s == nil {
		return constants.ArchiveStatusActive
	}
	return *s
}
