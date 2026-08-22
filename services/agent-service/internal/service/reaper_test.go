package service

import (
	"context"
	"errors"
	"testing"
	"time"

	factorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/agent-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
	"go.uber.org/mock/gomock"
)

// The reaper fails runs orphaned mid-flight, passing a cutoff ~stalledRunThreshold in the past.
func TestReapStalledRuns_FailsOrphanedRuns(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	// The cutoff must sit within a tight window around now-stalledRunThreshold so a healthy in-flight run is never reaped.
	cutoffMatcher := gomock.Cond(func(x any) bool {
		cutoff, ok := x.(time.Time)
		if !ok {
			return false
		}
		want := time.Now().Add(-stalledRunThreshold)
		return cutoff.After(want.Add(-time.Minute)) && cutoff.Before(want.Add(time.Minute))
	})

	runRepo := repositorymock.NewMockAgentRunRepo(ctrl)
	runRepo.EXPECT().
		ReapStalledRuns(gomock.Any(), cutoffMatcher, gomock.Any()).
		Return([]string{"agr_stuck_1", "agr_stuck_2"}, nil).
		Times(1)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAgentRunRepo().Return(runRepo).Times(1)

	s := &schedulerSvc{repos: factory}
	s.reapStalledRuns(context.Background())
}

// A reap error is swallowed (logged) so the poll loop keeps running.
func TestReapStalledRuns_SwallowsError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	runRepo := repositorymock.NewMockAgentRunRepo(ctrl)
	runRepo.EXPECT().
		ReapStalledRuns(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("db down"), "reap failed")).
		Times(1)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewAgentRunRepo().Return(runRepo).Times(1)

	s := &schedulerSvc{repos: factory}
	s.reapStalledRuns(context.Background()) // must not panic
}
