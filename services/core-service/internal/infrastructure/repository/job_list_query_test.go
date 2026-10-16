package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/pagination"
)

func TestBuildJobListQuery_OnlySentFiltersAreEmitted(t *testing.T) {
	query, args := buildJobListQuery(domain.ListJobsRepositoryParams{AccountID: "ac_1"}, pagination.DirectionForward, nil, "", 26)
	assert.Contains(t, query, "FORCE INDEX (job_account_created_idx) WHERE j.account_id = ? ORDER BY j.created_at DESC, j.job_id DESC LIMIT ?")
	assert.Equal(t, []any{"ac_1", int32(26)}, args)
}

func TestBuildJobListQuery_Filters(t *testing.T) {
	creator := "acu_1"
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	query, args := buildJobListQuery(domain.ListJobsRepositoryParams{
		ListJobsParams: domain.ListJobsParams{
			Types:    []constants.JobType{constants.JobTypeBulkCreate},
			Statuses: []constants.JobStatus{constants.JobStatusRunning, constants.JobStatusCanceled},
		},
		AccountID:   "ac_1",
		CreatedByID: &creator,
	}, pagination.DirectionBackward, &at, "jb_9", 11)

	assert.Contains(t, query, "j.created_by = ?")
	assert.Contains(t, query, "j.type IN (?,?)")
	assert.Contains(t, query, "(j.cancelled_at IS NULL AND j.completed_at IS NULL AND j.failed_at IS NULL AND j.started_at IS NOT NULL) OR j.cancelled_at IS NOT NULL")
	assert.Contains(t, query, "(j.created_at > ? OR (j.created_at = ? AND j.job_id > ?))")
	assert.True(t, strings.HasSuffix(query, "ORDER BY j.created_at ASC, j.job_id ASC LIMIT ?"))
	assert.Equal(t, []any{"ac_1", "acu_1", "bulk_create", "bulkcreate", at, at, "jb_9", int32(11)}, args)
}
