package mediator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

const (
	releaseAccountID = "acct_release"
	releaseStepID    = "prst_gated"
	releaseRunID     = "prru_1"
	releaseDeptID    = "dp_knitting"
)

var releaseScannedAt = time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)

type BatchReleaseMedTestSuite struct {
	suite.Suite
	ctrl          *gomock.Controller
	med           domain.BatchReleaseMed
	batchRepo     *repositorymock.MockBatchRepo
	policyRepo    *repositorymock.MockReleasePolicyRepo
	signatureRepo *repositorymock.MockBatchSignatureRepo
}

func (s *BatchReleaseMedTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.batchRepo = repositorymock.NewMockBatchRepo(s.ctrl)
	s.policyRepo = repositorymock.NewMockReleasePolicyRepo(s.ctrl)
	s.signatureRepo = repositorymock.NewMockBatchSignatureRepo(s.ctrl)

	repos := factorymock.NewMockRepoFactory(s.ctrl)
	repos.EXPECT().NewBatchRepo().Return(s.batchRepo).AnyTimes()
	repos.EXPECT().NewReleasePolicyRepo().Return(s.policyRepo).AnyTimes()
	repos.EXPECT().NewBatchSignatureRepo().Return(s.signatureRepo).AnyTimes()
	s.med = NewBatchReleaseMed(&BatchReleaseMedConfig{Repos: repos})
}

func TestBatchReleaseMedTestSuite(t *testing.T) {
	suite.Run(t, new(BatchReleaseMedTestSuite))
}

func gatedBatch(id string) *domain.BaseBatch {
	runID := releaseRunID
	scannedAt := releaseScannedAt
	departmentID := releaseDeptID
	return &domain.BaseBatch{
		ID:              id,
		DepartmentID:    &departmentID,
		ProductionStep:  &domain.LightProductionStep{ID: releaseStepID, Name: "Knitting"},
		ProductionRunID: &runID,
		ScannedAt:       &scannedAt,
	}
}

func releaseSignature(batchID string, capacity constants.SignatureCapacity, decision constants.BatchSignatureDecision, after time.Duration) *domain.BatchSignature {
	return &domain.BatchSignature{
		BatchID:          batchID,
		ProductionStepID: releaseStepID,
		Type:             constants.BatchSignatureTypeStepRelease,
		Decision:         decision,
		Capacity:         capacity,
		SignedAt:         releaseScannedAt.Add(after),
	}
}

func (s *BatchReleaseMedTestSuite) expectBatch(batch *domain.BaseBatch, requirement constants.ReleaseRequirement, enforcement constants.ReleaseEnforcement, signatures ...*domain.BatchSignature) {
	s.batchRepo.EXPECT().ListBasesByIDs(gomock.Any(), releaseAccountID, []string{batch.ID}).Return([]*domain.BaseBatch{batch}, nil)
	s.policyRepo.EXPECT().ListByDepartmentIDs(gomock.Any(), releaseAccountID, []string{releaseDeptID}).Return(map[string]*domain.ReleasePolicy{
		releaseDeptID: {DepartmentID: releaseDeptID, DepartmentName: "Knitting", Requirement: requirement, Enforcement: enforcement},
	}, nil)
	s.signatureRepo.EXPECT().ListForBatch(gomock.Any(), releaseAccountID, batch.ID).Return(signatures, nil).AnyTimes()
}

func (s *BatchReleaseMedTestSuite) TestLetsABatchLeaveAStepThatRequiresNothing() {
	batch := gatedBatch("bt_1")
	s.batchRepo.EXPECT().ListBasesByIDs(gomock.Any(), releaseAccountID, []string{batch.ID}).Return([]*domain.BaseBatch{batch}, nil)
	s.policyRepo.EXPECT().ListByDepartmentIDs(gomock.Any(), releaseAccountID, []string{releaseDeptID}).Return(map[string]*domain.ReleasePolicy{
		releaseDeptID: {DepartmentID: releaseDeptID, Requirement: constants.ReleaseRequirementNone},
	}, nil)

	s.Nil(s.med.CheckCanLeave(context.Background(), releaseAccountID, []string{batch.ID}))
}

func (s *BatchReleaseMedTestSuite) TestBlocksAnUnreleasedBatch() {
	batch := gatedBatch("bt_1")
	s.expectBatch(batch, constants.ReleaseRequirementSupervisor, constants.ReleaseEnforcementBlock)

	apiErr := s.med.CheckCanLeave(context.Background(), releaseAccountID, []string{batch.ID})

	s.Require().NotNil(apiErr)
	s.Equal(apierror.ErrorCodeBatchReleaseRequired, apiErr.Code)
}

func (s *BatchReleaseMedTestSuite) TestWarnLetsAnUnreleasedBatchThrough() {
	batch := gatedBatch("bt_1")
	s.expectBatch(batch, constants.ReleaseRequirementSupervisor, constants.ReleaseEnforcementWarn)

	s.Nil(s.med.CheckCanLeave(context.Background(), releaseAccountID, []string{batch.ID}))
}

func (s *BatchReleaseMedTestSuite) TestLetsAReleasedBatchLeave() {
	batch := gatedBatch("bt_1")
	s.expectBatch(batch, constants.ReleaseRequirementQuality, constants.ReleaseEnforcementBlock,
		releaseSignature(batch.ID, constants.SignatureCapacityQuality, constants.BatchSignatureDecisionApproved, time.Minute))

	s.Nil(s.med.CheckCanLeave(context.Background(), releaseAccountID, []string{batch.ID}))
}

func (s *BatchReleaseMedTestSuite) TestHoldsARejectedBatchEvenWhenTheStepOnlyWarns() {
	batch := gatedBatch("bt_1")
	s.expectBatch(batch, constants.ReleaseRequirementQuality, constants.ReleaseEnforcementWarn,
		releaseSignature(batch.ID, constants.SignatureCapacityQuality, constants.BatchSignatureDecisionRejected, 2*time.Minute),
		releaseSignature(batch.ID, constants.SignatureCapacityQuality, constants.BatchSignatureDecisionApproved, time.Minute))

	apiErr := s.med.CheckCanLeave(context.Background(), releaseAccountID, []string{batch.ID})

	s.Require().NotNil(apiErr)
	s.Equal(apierror.ErrorCodeBatchOnHold, apiErr.Code)
}

func (s *BatchReleaseMedTestSuite) TestIgnoresAReleaseFromBeforeTheBatchWasLastScanned() {
	batch := gatedBatch("bt_1")
	s.expectBatch(batch, constants.ReleaseRequirementSupervisor, constants.ReleaseEnforcementBlock,
		releaseSignature(batch.ID, constants.SignatureCapacitySupervisor, constants.BatchSignatureDecisionApproved, -time.Minute))

	apiErr := s.med.CheckCanLeave(context.Background(), releaseAccountID, []string{batch.ID})

	s.Require().NotNil(apiErr)
	s.Equal(apierror.ErrorCodeBatchReleaseRequired, apiErr.Code)
}

func (s *BatchReleaseMedTestSuite) TestFirstArticleNeedsBothCapacitiesAcrossTheRun() {
	batch := gatedBatch("bt_2")
	s.expectBatch(batch, constants.ReleaseRequirementFirstArticle, constants.ReleaseEnforcementBlock)
	// Only the supervisor has signed the run's first batch so far.
	s.signatureRepo.EXPECT().ListRunStepReleases(gomock.Any(), releaseAccountID, releaseRunID, releaseStepID).Return([]*domain.BatchSignature{
		releaseSignature("bt_1", constants.SignatureCapacitySupervisor, constants.BatchSignatureDecisionApproved, time.Minute),
	}, nil)

	apiErr := s.med.CheckCanLeave(context.Background(), releaseAccountID, []string{batch.ID})

	s.Require().NotNil(apiErr)
	s.Equal(apierror.ErrorCodeBatchReleaseRequired, apiErr.Code)
}

func (s *BatchReleaseMedTestSuite) TestFirstArticleSignedOnOneBatchCoversTheRun() {
	batch := gatedBatch("bt_2")
	s.expectBatch(batch, constants.ReleaseRequirementFirstArticle, constants.ReleaseEnforcementBlock)
	s.signatureRepo.EXPECT().ListRunStepReleases(gomock.Any(), releaseAccountID, releaseRunID, releaseStepID).Return([]*domain.BatchSignature{
		releaseSignature("bt_1", constants.SignatureCapacityQuality, constants.BatchSignatureDecisionApproved, 2*time.Minute),
		releaseSignature("bt_1", constants.SignatureCapacitySupervisor, constants.BatchSignatureDecisionApproved, time.Minute),
	}, nil)

	s.Nil(s.med.CheckCanLeave(context.Background(), releaseAccountID, []string{batch.ID}))
}

func (s *BatchReleaseMedTestSuite) TestLineageCountsUpstreamBatchesAWarnStepLetThrough() {
	upstream := gatedBatch("bt_up")
	s.batchRepo.EXPECT().FindBatchFlow(gomock.Any(), releaseAccountID, "bt_final").Return([]domain.BatchFlowNode{
		{Batch: domain.Batch{ID: "bt_final"}, InputBatchIDs: []string{"bt_up"}},
		{Batch: domain.Batch{ID: "bt_up"}, OutputBatchIDs: []string{"bt_final"}},
	}, nil)
	s.expectBatch(upstream, constants.ReleaseRequirementSupervisor, constants.ReleaseEnforcementWarn)

	apiErr := s.med.CheckLineageReleased(context.Background(), releaseAccountID, "bt_final")

	s.Require().NotNil(apiErr)
	s.Equal(apierror.ErrorCodeBatchReleaseRequired, apiErr.Code)
}
