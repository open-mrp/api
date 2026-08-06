package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	servicemock "github.com/augno/api/services/core-service/internal/domain/mock/service"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/excel"

	"github.com/stretchr/testify/suite"
	"github.com/xuri/excelize/v2"
	"go.uber.org/mock/gomock"
)

// keeps what was uploaded instead of discarding it, so a test can assert on the bytes
// that actually reached object storage rather than the ones the builder returned.
type capturingObjectStore struct {
	bucket      string
	key         string
	body        []byte
	contentType string
	uploads     int
	uploadErr   *apierror.APIError
}

func (c *capturingObjectStore) Upload(_ context.Context, bucket, key string, body io.Reader, contentType string) *apierror.APIError {
	c.uploads++
	if c.uploadErr != nil {
		return c.uploadErr
	}
	read, err := io.ReadAll(body)
	if err != nil {
		return apierror.NewInternalError(err, "fake store could not read the body")
	}
	c.bucket, c.key, c.body, c.contentType = bucket, key, read, contentType
	return nil
}

func (c *capturingObjectStore) GetPresignedURL(_ context.Context, bucket, key string, _ time.Duration) (string, *apierror.APIError) {
	return "https://signed.test/" + bucket + "/" + key, nil
}

func (c *capturingObjectStore) FileExists(context.Context, string, string) (bool, *apierror.APIError) {
	return true, nil
}

func (c *capturingObjectStore) Copy(context.Context, string, string, string) *apierror.APIError {
	return nil
}

func (c *capturingObjectStore) Delete(context.Context, string, string) *apierror.APIError {
	return nil
}

func (c *capturingObjectStore) Get(context.Context, string, string) ([]byte, *apierror.APIError) {
	return c.body, nil
}

func (c *capturingObjectStore) GetPresignedPutURL(_ context.Context, bucket, key, _ string, _ time.Duration) (string, *apierror.APIError) {
	return "https://signed.test/put/" + bucket + "/" + key, nil
}

type ExportRunnerTestSuite struct {
	suite.Suite
	store         *capturingObjectStore
	delivery      ExportDelivery
	repoFactory   *factorymock.MockRepoFactory
	jobSvc        *servicemock.MockJobSvc
	jobSvcFactory *factorymock.MockJobSvcFactory
	ctrl          *gomock.Controller
}

func (suite *ExportRunnerTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.store = &capturingObjectStore{}
	suite.delivery = NewExportDelivery(suite.store, "augno-exports-test")
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.jobSvc = servicemock.NewMockJobSvc(suite.ctrl)
	suite.jobSvcFactory = factorymock.NewMockJobSvcFactory(suite.ctrl)
	suite.jobSvcFactory.EXPECT().Build(gomock.Any()).Return(suite.jobSvc).AnyTimes()
}

func (suite *ExportRunnerTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestExportRunnerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ExportRunnerTestSuite))
}

// builds a runner whose one registered resource renders the given file
func (suite *ExportRunnerTestSuite) runner(slug string, export *domain.Export, buildErr *apierror.APIError) *ExportRunner {
	return NewExportRunner(&ExportRunnerConfig{
		Repos:         suite.repoFactory,
		JobSvcFactory: suite.jobSvcFactory,
		Delivery:      suite.delivery,
		Builders: map[string]domain.ExportBuilder{
			slug: func(context.Context, string, json.RawMessage) (*domain.Export, *apierror.APIError) {
				return export, buildErr
			},
		},
	})
}

func exportJob(jobID, accountID, slug string) *domain.Job {
	items, _ := json.Marshal(exportJobPayload{Slug: slug, Filters: json.RawMessage(`{}`)})
	return &domain.Job{
		ID:        jobID,
		Type:      "export",
		AccountID: &accountID,
		JobItems:  items,
	}
}

// a workbook with one readable row, so the upload can be parsed rather than compared
func testWorkbook(suite *suite.Suite) []byte {
	body, err := excel.Build(excel.Spec{Sheets: []excel.Sheet{{
		Name:    "Materials",
		Columns: []excel.ColumnSpec{{Header: "SKU", Key: "sku", Width: 10}},
		Rows:    []excel.Row{{"sku": "YRN-1"}},
	}}})
	suite.Require().NoError(err)
	return body
}

func (suite *ExportRunnerTestSuite) TestRender_UploadsTheWorkbookAndCompletesTheJob() {
	ctx := internalIdentityCtx("ac_test123")
	startedAt := time.Date(2026, time.July, 31, 9, 0, 0, 0, time.UTC)
	job := exportJob("job_1", "ac_test123", "materials")
	body := testWorkbook(&suite.Suite)

	suite.jobSvc.EXPECT().GetJobForExecution(gomock.Any(), "job_1").Return(job, nil)
	suite.jobSvc.EXPECT().StartJob(gomock.Any(), gomock.Any()).Return(startedAt, nil)
	suite.jobSvc.EXPECT().CompleteJob(gomock.Any(), gomock.Any()).Return(nil)

	runner := suite.runner("materials", &domain.Export{Body: body, ContentType: excel.ContentType}, nil)
	apiErr := runner.Render(ctx, domain.BulkOperationJobEvent{JobID: "job_1"})
	suite.Require().Nil(apiErr)

	suite.Equal("augno-exports-test", suite.store.bucket)
	suite.Equal("exports/ac_test123/materials/job_1/materials_export_07-31-2026.xlsx", suite.store.key)
	suite.Equal(excel.ContentType, suite.store.contentType)

	// The bytes that reached storage are the workbook, not a copy of something else.
	f, err := excelize.OpenReader(bytes.NewReader(suite.store.body))
	suite.Require().NoError(err)
	suite.T().Cleanup(func() { _ = f.Close() })
	rows, err := f.GetRows("Materials")
	suite.Require().NoError(err)
	suite.Equal([][]string{{"SKU"}, {"YRN-1"}}, rows)
}

// The reader has to land on the object the worker wrote, and nothing records where that
// was — so the two derivations have to agree from the job alone.
func (suite *ExportRunnerTestSuite) TestRender_UploadsWhereTheReaderWillLookForIt() {
	ctx := internalIdentityCtx("ac_test123")
	startedAt := time.Date(2026, time.July, 31, 9, 0, 0, 0, time.UTC)
	job := exportJob("job_1", "ac_test123", "materials")

	suite.jobSvc.EXPECT().GetJobForExecution(gomock.Any(), "job_1").Return(job, nil)
	suite.jobSvc.EXPECT().StartJob(gomock.Any(), gomock.Any()).Return(startedAt, nil)
	suite.jobSvc.EXPECT().CompleteJob(gomock.Any(), gomock.Any()).Return(nil)

	runner := suite.runner("materials", &domain.Export{Body: testWorkbook(&suite.Suite), ContentType: excel.ContentType}, nil)
	suite.Require().Nil(runner.Render(ctx, domain.BulkOperationJobEvent{JobID: "job_1"}))

	// What the reader would sign, derived from the job alone.
	completed := exportJob("job_1", "ac_test123", "materials")
	completed.StartedAt = &startedAt
	url, apiErr := exportKeyFor(completed)
	suite.Require().Nil(apiErr)
	suite.Equal(suite.store.key, url)
}

// The inbox de-dupes redeliveries, but a replay outliving that record must not rebuild a
// file and re-settle a job that is already done.
func (suite *ExportRunnerTestSuite) TestRender_SkipsAJobThatAlreadySettled() {
	ctx := internalIdentityCtx("ac_test123")
	completedAt := time.Now().UTC()
	job := exportJob("job_1", "ac_test123", "materials")
	job.CompletedAt = &completedAt

	suite.jobSvc.EXPECT().GetJobForExecution(gomock.Any(), "job_1").Return(job, nil)

	runner := suite.runner("materials", &domain.Export{Body: testWorkbook(&suite.Suite)}, nil)
	suite.Require().Nil(runner.Render(ctx, domain.BulkOperationJobEvent{JobID: "job_1"}))
	suite.Zero(suite.store.uploads, "a settled job must not be rendered again")
}

// A render that fails leaves the job failed and nothing in storage, so a completed job
// always has a file behind it.
func (suite *ExportRunnerTestSuite) TestRender_FailsTheJobWhenTheBuildFails() {
	ctx := internalIdentityCtx("ac_test123")
	job := exportJob("job_1", "ac_test123", "materials")

	suite.jobSvc.EXPECT().GetJobForExecution(gomock.Any(), "job_1").Return(job, nil)
	suite.jobSvc.EXPECT().StartJob(gomock.Any(), gomock.Any()).Return(time.Now().UTC(), nil)
	suite.jobSvc.EXPECT().FailJob(gomock.Any(), gomock.Any()).Times(1)

	runner := suite.runner("materials", nil, apierror.NewInternalError(nil, "the query blew up"))
	suite.NotNil(runner.Render(ctx, domain.BulkOperationJobEvent{JobID: "job_1"}))
	suite.Zero(suite.store.uploads)
}

// An upload failure must not complete the job: the link would point at nothing.
func (suite *ExportRunnerTestSuite) TestRender_FailsTheJobWhenTheUploadFails() {
	ctx := internalIdentityCtx("ac_test123")
	job := exportJob("job_1", "ac_test123", "materials")
	suite.store.uploadErr = apierror.NewInternalError(nil, "bucket is missing")

	suite.jobSvc.EXPECT().GetJobForExecution(gomock.Any(), "job_1").Return(job, nil)
	suite.jobSvc.EXPECT().StartJob(gomock.Any(), gomock.Any()).Return(time.Now().UTC(), nil)
	suite.jobSvc.EXPECT().FailJob(gomock.Any(), gomock.Any()).Times(1)

	runner := suite.runner("materials", &domain.Export{Body: testWorkbook(&suite.Suite)}, nil)
	suite.NotNil(runner.Render(ctx, domain.BulkOperationJobEvent{JobID: "job_1"}))
}

// A slug with no registered builder is a wiring mistake, and it must surface on the job
// rather than leaving it running forever.
func (suite *ExportRunnerTestSuite) TestRender_FailsTheJobForAnUnregisteredResource() {
	ctx := internalIdentityCtx("ac_test123")
	job := exportJob("job_1", "ac_test123", "widgets")

	suite.jobSvc.EXPECT().GetJobForExecution(gomock.Any(), "job_1").Return(job, nil)
	suite.jobSvc.EXPECT().FailJob(gomock.Any(), gomock.Any()).Times(1)

	runner := suite.runner("materials", &domain.Export{Body: testWorkbook(&suite.Suite)}, nil)
	suite.NotNil(runner.Render(ctx, domain.BulkOperationJobEvent{JobID: "job_1"}))
	suite.Zero(suite.store.uploads)
}

// Only a completed export has a file; everything else must read back as a plain job
// carrying no download.
func (suite *ExportRunnerTestSuite) TestDownloadURL_OnlySignsACompletedExport() {
	startedAt := time.Date(2026, time.July, 31, 9, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(time.Minute)
	svc := NewExportSvc(&ExportSvcConfig{Delivery: suite.delivery})

	running := exportJob("job_1", "ac_test123", "materials")
	running.StartedAt = &startedAt

	bulk := exportJob("job_2", "ac_test123", "materials")
	bulk.Type = "bulkupsert"
	bulk.StartedAt = &startedAt
	bulk.CompletedAt = &completedAt

	for _, job := range []*domain.Job{running, bulk} {
		url, apiErr := svc.DownloadURL(context.Background(), job)
		suite.Require().Nil(apiErr)
		suite.Empty(url, "%s job should not carry a download", job.Type)
	}

	done := exportJob("job_3", "ac_test123", "unit_groups")
	done.StartedAt = &startedAt
	done.CompletedAt = &completedAt

	url, apiErr := svc.DownloadURL(context.Background(), done)
	suite.Require().Nil(apiErr)
	suite.Contains(url, "exports/ac_test123/unit_groups/job_3/unit_groups_export_07-31-2026.xlsx")
}
