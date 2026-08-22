package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/DATA-DOG/go-sqlmock"
)

// A job is created with results and errors empty and only fills them in when it
// settles, so reading one back right after creating it scans NULL out of both
// columns. json.RawMessage cannot do that — database/sql special cases *[]byte but
// not named types of it — which is why those columns are mapped to
// db.NullableRawMessage. This drives the real database/sql scan path, so it fails if
// that mapping is ever dropped.
func TestJobRepo_Get_ReadsAJobWhoseJSONColumnsAreNull(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "type", "resource_type", "account_id", "created_by",
			"job_items", "results", "error", "errors",
			"started_at", "completed_at", "failed_at", "cancelled_at",
			"created_at", "updated_at",
		}).AddRow(
			"jo_test", "bulk_create", "production_run", "ac_test", nil,
			[]byte(`{"runs":[]}`), nil, nil, nil,
			nil, nil, nil, nil,
			now, now,
		))

	repo := NewJobRepo(sqlc.New(db))

	job, apiErr := repo.Get(context.Background(), "jo_test", "ac_test")

	if apiErr != nil {
		t.Fatalf("expected a freshly created job to read back, got: %v", apiErr)
	}
	// A NULL column decodes to a nil list, which is what distinguishes a job that has
	// recorded no results from one that ran and wrote none (an empty list).
	if job.Results != nil {
		t.Errorf("expected no results on an unsettled job, got %v", job.Results)
	}
	if job.Error != nil {
		t.Errorf("expected no error on an unsettled job, got %v", job.Error)
	}
	if string(job.JobItems) != `{"runs":[]}` {
		t.Errorf("job items should survive the read, got %q", string(job.JobItems))
	}
}

// Encoding the row outcomes is the repository's job — the layers above it hand over
// domain values — so this pins the JSON that column actually holds and the decode back
// out of it. The empty-but-present list is the case that matters: it has to stay
// distinguishable from NULL, because it is how a job that ran and wrote nothing differs
// from one that has not recorded results at all.
func TestJobRepo_Get_DecodesRowOutcomes(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "type", "resource_type", "account_id", "created_by",
			"job_items", "results", "error", "errors",
			"started_at", "completed_at", "failed_at", "cancelled_at",
			"created_at", "updated_at",
		}).AddRow(
			"jo_test", "bulk_upsert", "unit", "ac_test", nil,
			[]byte(`{}`),
			[]byte(`{"rows":[{"index":0,"status":"created","resource_type":"unit","id":"un_1","sub_resources":[{"resource_type":"batch","id":"ba_1"}]},{"index":1,"status":"failed","error":{"code":"validation_failed","type":"invalid_request_error","message":"bad row","param":"name"}}],"truncated":true}`),
			[]byte(`{"code":"internal_error","type":"api_error","message":"the whole batch fell over"}`),
			nil,
			nil, nil, nil, nil,
			now, now,
		))

	repo := NewJobRepo(sqlc.New(db))

	job, apiErr := repo.Get(context.Background(), "jo_test", "ac_test")
	if apiErr != nil {
		t.Fatalf("expected the job to read back, got: %v", apiErr)
	}

	if job.ResourceType != constants.ObjectTypeUnit {
		t.Errorf("resource type did not decode: %q", job.ResourceType)
	}
	if !job.ResultsTruncated {
		t.Error("a trimmed record must read back as trimmed")
	}
	if len(job.Results) != 2 {
		t.Fatalf("expected both rows, got %v", job.Results)
	}

	written := job.Results[0]
	if written.Index != 0 || written.ID != "un_1" || written.Status != constants.JobResultStatusCreated {
		t.Errorf("written row did not decode: %+v", written)
	}
	if written.ResourceType != constants.ObjectTypeUnit {
		t.Errorf("the row must name what it produced: %+v", written)
	}
	if len(written.SubResources) != 1 || written.SubResources[0].ID != "ba_1" || written.SubResources[0].ResourceType != constants.ObjectTypeBatch {
		t.Errorf("sub-resources did not decode: %+v", written.SubResources)
	}

	failed := job.Results[1]
	if failed.Index != 1 || failed.Status != constants.JobResultStatusFailed {
		t.Errorf("rejected row did not decode: %+v", failed)
	}
	if failed.Error == nil || failed.Error.Code != apierror.ErrorCodeValidationFailed || failed.Error.Message != "bad row" {
		t.Errorf("the row's error did not decode: %+v", failed.Error)
	}
	// A row's own failure never doubles as the job's — the two are different scopes.
	if job.Error == nil || job.Error.Message != "the whole batch fell over" {
		t.Errorf("the whole-job error did not decode: %+v", job.Error)
	}
}

// A job raised before the outcomes merged stored its written rows as a bare array and
// its failures in a separate column. Those settle within minutes, so this only has to
// cover the ones in flight across the deploy — but for those, one entry per submitted
// row still has to come back. Delete with the legacy decode it drives.
func TestJobRepo_Get_FoldsALegacyJobsSeparateErrorsIntoItsRows(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "type", "resource_type", "account_id", "created_by",
			"job_items", "results", "error", "errors",
			"started_at", "completed_at", "failed_at", "cancelled_at",
			"created_at", "updated_at",
		}).AddRow(
			"jo_test", "bulkupsert", "unit", "ac_test", nil,
			[]byte(`{}`),
			[]byte(`[{"Index":0,"ID":"un_1","Action":"created","SubResourceIDs":["ba_1"]}]`),
			nil,
			[]byte(`[{"index":1,"error":{"code":"validation_failed","type":"invalid_request_error","message":"bad row","param":"name"}},{"error":{"code":"internal_error","type":"api_error","message":"the whole batch fell over"}}]`),
			nil, nil, nil, nil,
			now, now,
		))

	repo := NewJobRepo(sqlc.New(db))

	job, apiErr := repo.Get(context.Background(), "jo_test", "ac_test")
	if apiErr != nil {
		t.Fatalf("expected the legacy job to read back, got: %v", apiErr)
	}

	if len(job.Results) != 2 {
		t.Fatalf("the legacy errors must fold in as rows, got %v", job.Results)
	}
	if job.Results[0].Status != constants.JobResultStatusCreated || job.Results[0].ID != "un_1" {
		t.Errorf("the legacy action must become the row's status: %+v", job.Results[0])
	}
	// The legacy row named only ids, so the kind comes from the job's own resource type.
	if len(job.Results[0].SubResources) != 1 || job.Results[0].SubResources[0].ResourceType != constants.ObjectTypeUnit {
		t.Errorf("legacy sub-resources did not decode: %+v", job.Results[0].SubResources)
	}
	if job.Results[1].Index != 1 || job.Results[1].Status != constants.JobResultStatusFailed {
		t.Errorf("the legacy row error must become a failed row: %+v", job.Results[1])
	}
	// The index-less legacy entry named no row, so it settles on the job.
	if job.Error == nil || job.Error.Message != "the whole batch fell over" {
		t.Errorf("the legacy batch error must become the job's error: %+v", job.Error)
	}
}

// The update's guard and Job.IsTerminal have to name the same states, and nothing in
// the type system couples them: the guard is SQL and IsTerminal is Go. They drifted
// once — the guard also required failed_at IS NULL, which froze a failed job against
// every later transition even though IsTerminal deliberately reports it as retryable,
// so a redelivery passed the Go check and was then refused by the query. This captures
// the SQL the driver actually receives and pins the two together.
func TestJobRepo_Update_GuardsOnTheTerminalStatesOnly(t *testing.T) {
	t.Parallel()

	var captured string
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(
		sqlmock.QueryMatcherFunc(func(_, actualSQL string) error {
			captured = actualSQL
			return nil
		}),
	))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewJobRepo(sqlc.New(db))
	completedAt := time.Now().UTC()
	if _, apiErr := repo.Update(context.Background(), domain.UpdateJobRepositoryParams{
		JobID:       "jo_test",
		AccountID:   "ac_test",
		CompletedAt: &completedAt,
	}); apiErr != nil {
		t.Fatalf("update failed: %v", apiErr)
	}

	// Every state Job.IsTerminal calls terminal must be guarded, and no other, so the
	// two cannot disagree about what settles a job.
	for _, guarded := range []string{"completed_at IS NULL", "cancelled_at IS NULL"} {
		if !strings.Contains(captured, guarded) {
			t.Errorf("the update must refuse a settled job: expected the guard to include %q\n%s", guarded, captured)
		}
	}
	if strings.Contains(captured, "failed_at IS NULL") {
		t.Errorf("a failed job is retryable (see Job.IsTerminal), so the guard must not freeze it\n%s", captured)
	}
}

// An empty results list is not the same as no results, and the difference has to
// survive the round trip through the column.
func TestJobRepo_Update_EncodesAnEmptyResultsListDistinctlyFromNone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		results []domain.RowResult
		want    any
	}{
		{
			name:    "no results leaves the column alone",
			results: nil,
			want:    nil,
		},
		{
			name:    "an empty list records an empty array",
			results: []domain.RowResult{},
			want:    []byte(`{"rows":[],"truncated":false}`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create sqlmock: %v", err)
			}
			defer db.Close()

			mock.ExpectExec("UPDATE").
				WithArgs(tc.want, nil, nil, sqlmock.AnyArg(), nil, nil, "jo_test", "ac_test").
				WillReturnResult(sqlmock.NewResult(0, 1))

			repo := NewJobRepo(sqlc.New(db))
			completedAt := time.Now().UTC()

			if _, apiErr := repo.Update(context.Background(), domain.UpdateJobRepositoryParams{
				JobID:       "jo_test",
				AccountID:   "ac_test",
				Results:     tc.results,
				CompletedAt: &completedAt,
			}); apiErr != nil {
				t.Fatalf("update failed: %v", apiErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("the results column was not written as expected: %v", err)
			}
		})
	}
}
