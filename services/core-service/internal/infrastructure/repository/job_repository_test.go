package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

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
			"id", "type", "account_id", "created_by",
			"created_by_name", "created_by_username", "created_by_email",
			"job_items", "results", "errors", "error_summary",
			"started_at", "completed_at", "failed_at", "cancelled_at",
			"created_at", "updated_at",
		}).AddRow(
			"jo_test", "bulkcreate", "ac_test", nil,
			nil, nil, nil,
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
	if job.Errors != nil {
		t.Errorf("expected no errors on an unsettled job, got %v", job.Errors)
	}
	if string(job.JobItems) != `{"runs":[]}` {
		t.Errorf("job items should survive the read, got %q", string(job.JobItems))
	}
}

// Encoding the results and errors lists is the repository's job — the layers above it
// hand over domain values — so this pins the JSON those columns actually hold and the
// decode back out of it. The empty-but-present results list is the case that matters:
// it has to stay distinguishable from NULL, because it is how a job that ran and wrote
// nothing differs from one that has not recorded results at all.
func TestJobRepo_Get_DecodesResultsAndErrors(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "type", "account_id", "created_by",
			"created_by_name", "created_by_username", "created_by_email",
			"job_items", "results", "errors", "error_summary",
			"started_at", "completed_at", "failed_at", "cancelled_at",
			"created_at", "updated_at",
		}).AddRow(
			"jo_test", "bulkupsert", "ac_test", nil,
			nil, nil, nil,
			[]byte(`{}`),
			[]byte(`[{"Index":0,"ID":"un_1","Action":"created","SubResourceIDs":["ba_1"]}]`),
			[]byte(`[{"Index":1,"Error":{"code":"validation_failed","type":"invalid_request_error","message":"bad row","param":"name","doc_url":null,"is_transient":false,"quota":null,"request_log_url":null}}]`),
			nil,
			nil, nil, nil, nil,
			now, now,
		))

	repo := NewJobRepo(sqlc.New(db))

	job, apiErr := repo.Get(context.Background(), "jo_test", "ac_test")
	if apiErr != nil {
		t.Fatalf("expected the job to read back, got: %v", apiErr)
	}

	if len(job.Results) != 1 {
		t.Fatalf("expected one result, got %v", job.Results)
	}
	got := job.Results[0]
	if got.Index != 0 || got.ID != "un_1" || got.Action != constants.JobResultActionCreated {
		t.Errorf("result did not decode: %+v", got)
	}
	if len(got.SubResourceIDs) != 1 || got.SubResourceIDs[0] != "ba_1" {
		t.Errorf("sub-resource ids did not decode: %+v", got.SubResourceIDs)
	}

	if len(job.Errors) != 1 {
		t.Fatalf("expected one error entry, got %v", job.Errors)
	}
	gotErr := job.Errors[0]
	if gotErr.Index == nil || *gotErr.Index != 1 {
		t.Errorf("error index did not decode: %+v", gotErr.Index)
	}
	if gotErr.Error.Code != apierror.ErrorCodeValidationFailed || gotErr.Error.Message != "bad row" {
		t.Errorf("error object did not decode: %+v", gotErr.Error)
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
			want:    []byte(`[]`),
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
				WithArgs(tc.want, nil, nil, nil, sqlmock.AnyArg(), nil, nil, "jo_test", "ac_test").
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
