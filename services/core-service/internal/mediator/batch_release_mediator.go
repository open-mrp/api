package mediator

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var batchReleaseMedTracer = tracing.GetTracer("core-service.batch_release_mediator")

type batchReleaseMedImpl struct {
	repos domain.RepoFactory
}

type BatchReleaseMedConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
}

func (c *BatchReleaseMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("batch release mediator: repos is required")
	}
	return nil
}

func NewBatchReleaseMed(config *BatchReleaseMedConfig) domain.BatchReleaseMed {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &batchReleaseMedImpl{repos: config.Repos}
}

// CheckCanLeave refuses a scan that would take batches on before the sign-off their departments require.
//
//  1. Read the batches and the release policies of their departments; batches in departments requiring none pass.
//  2. Work out each remaining batch's release state from its signatures (see ReleaseState).
//  3. Refuse a batch on hold, whatever its department's enforcement.
//  4. Refuse an unreleased batch in a department that blocks; let it through in one that only warns.
func (m *batchReleaseMedImpl) CheckCanLeave(ctx context.Context, accountID string, batchIDs []string) *apierror.APIError {
	ctx, span := batchReleaseMedTracer.Start(ctx, "mediator.batch_release.check_can_leave")
	defer span.End()

	gated, policies, apiErr := m.gatedBatches(ctx, accountID, batchIDs)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	for _, batch := range gated {
		policy := policies[*batch.DepartmentID]
		released, onHold, apiErr := m.ReleaseState(ctx, accountID, batch, policy)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		switch {
		case onHold:
			return tracing.Trace(span, apierror.NewBatchOnHoldError(fmt.Sprintf("Batch %s was rejected in %s and can't move on until it is signed off again.", batch.ID, policy.DepartmentName)))
		case !released && policy.Enforcement == constants.ReleaseEnforcementBlock:
			return tracing.Trace(span, apierror.NewBatchReleaseRequiredError(fmt.Sprintf("Batch %s needs a %s sign-off in %s before it can move on.", batch.ID, requirementLabel(policy.Requirement), policy.DepartmentName)))
		}
	}
	return nil
}

// CheckLineageReleased refuses a final release while any batch upstream of this one still lacks a
// sign-off its department requires, including the ones a warn department let through.
//
//  1. Walk the batch's flow upstream from it.
//  2. Work out the release state of every upstream batch in a department that requires a sign-off.
//  3. Refuse when any of them is unreleased or on hold, saying how many.
func (m *batchReleaseMedImpl) CheckLineageReleased(ctx context.Context, accountID, batchID string) *apierror.APIError {
	ctx, span := batchReleaseMedTracer.Start(ctx, "mediator.batch_release.check_lineage_released")
	defer span.End()

	flow, apiErr := m.repos.NewBatchRepo().FindBatchFlow(ctx, accountID, batchID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	inputs := make(map[string][]string, len(flow))
	for _, node := range flow {
		inputs[node.Batch.ID] = node.InputBatchIDs
	}
	var upstream []string
	seen := map[string]bool{batchID: true}
	queue := append([]string(nil), inputs[batchID]...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		upstream = append(upstream, id)
		queue = append(queue, inputs[id]...)
	}

	gated, policies, apiErr := m.gatedBatches(ctx, accountID, upstream)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	missing := 0
	for _, batch := range gated {
		released, onHold, apiErr := m.ReleaseState(ctx, accountID, batch, policies[*batch.DepartmentID])
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		if !released || onHold {
			missing++
		}
	}
	if missing > 0 {
		plural := "es"
		if missing == 1 {
			plural = ""
		}
		return tracing.Trace(span, apierror.NewBatchReleaseRequiredError(fmt.Sprintf("%d earlier batch%s in this batch's production still need a sign-off. Sign them off before the final release.", missing, plural)))
	}
	return nil
}

// ReleaseState reports whether a batch has the sign-off its department requires, and whether it is on hold.
//
//  1. Count only release signatures made at the batch's current step since it was scanned there.
//  2. In a first-article department, the latest supervisor and the latest quality signature any batch of the run received at the step must both approve; the batch is on hold only when its own latest release was a rejection.
//  3. Otherwise the batch's latest release decides: approved is released, rejected is on hold, none is unreleased.
func (m *batchReleaseMedImpl) ReleaseState(ctx context.Context, accountID string, batch *domain.BaseBatch, policy *domain.ReleasePolicy) (bool, bool, *apierror.APIError) {
	ctx, span := batchReleaseMedTracer.Start(ctx, "mediator.batch_release.release_state")
	defer span.End()

	if policy == nil || policy.Requirement == constants.ReleaseRequirementNone {
		return true, false, nil
	}
	if batch.ProductionStep == nil || batch.ScannedAt == nil {
		return false, false, nil
	}

	signatures, apiErr := m.repos.NewBatchSignatureRepo().ListForBatch(ctx, accountID, batch.ID)
	if apiErr != nil {
		return false, false, tracing.Trace(span, apiErr)
	}
	own := currentReleases(signatures, batch)
	onHold := len(own) > 0 && own[0].Decision == constants.BatchSignatureDecisionRejected

	if policy.Requirement != constants.ReleaseRequirementFirstArticle {
		return len(own) > 0 && own[0].Decision == constants.BatchSignatureDecisionApproved, onHold, nil
	}

	// A batch that never belonged to a run has only its own signatures to go on.
	runReleases := own
	if batch.ProductionRunID != nil {
		runReleases, apiErr = m.repos.NewBatchSignatureRepo().ListRunStepReleases(ctx, accountID, *batch.ProductionRunID, batch.ProductionStep.ID)
		if apiErr != nil {
			return false, false, tracing.Trace(span, apiErr)
		}
	}
	return latestApproves(runReleases, constants.SignatureCapacitySupervisor) &&
		latestApproves(runReleases, constants.SignatureCapacityQuality), onHold, nil
}

// gatedBatches reads the batches and keeps those in a department that requires a sign-off, with those
// departments' policies.
func (m *batchReleaseMedImpl) gatedBatches(ctx context.Context, accountID string, batchIDs []string) ([]*domain.BaseBatch, map[string]*domain.ReleasePolicy, *apierror.APIError) {
	batches, apiErr := m.repos.NewBatchRepo().ListBasesByIDs(ctx, accountID, batchIDs)
	if apiErr != nil {
		return nil, nil, apiErr
	}
	var departmentIDs []string
	for _, batch := range batches {
		if batch.DepartmentID != nil {
			departmentIDs = append(departmentIDs, *batch.DepartmentID)
		}
	}
	policies, apiErr := m.repos.NewReleasePolicyRepo().ListByDepartmentIDs(ctx, accountID, departmentIDs)
	if apiErr != nil {
		return nil, nil, apiErr
	}
	var gated []*domain.BaseBatch
	for _, batch := range batches {
		if batch.DepartmentID == nil || batch.ProductionStep == nil {
			continue
		}
		if policy := policies[*batch.DepartmentID]; policy != nil && policy.Requirement != constants.ReleaseRequirementNone {
			gated = append(gated, batch)
		}
	}
	return gated, policies, nil
}

// currentReleases keeps the release signatures a batch received at its current step since it was last
// scanned there, newest first. Ones from before an undo and rescan speak for a scan that no longer stands.
func currentReleases(signatures []*domain.BatchSignature, batch *domain.BaseBatch) []*domain.BatchSignature {
	var current []*domain.BatchSignature
	for _, signature := range signatures {
		if !signature.Type.IsRelease() || signature.ProductionStepID != batch.ProductionStep.ID || signature.SignedAt.Before(*batch.ScannedAt) {
			continue
		}
		current = append(current, signature)
	}
	return current
}

// latestApproves reports whether the newest signature made in the capacity approves.
func latestApproves(signatures []*domain.BatchSignature, capacity constants.SignatureCapacity) bool {
	for _, signature := range signatures {
		if signature.Capacity == capacity {
			return signature.Decision == constants.BatchSignatureDecisionApproved
		}
	}
	return false
}

func requirementLabel(requirement constants.ReleaseRequirement) string {
	switch requirement {
	case constants.ReleaseRequirementSupervisor:
		return "supervisor"
	case constants.ReleaseRequirementQuality:
		return "quality"
	case constants.ReleaseRequirementFirstArticle:
		return "first-article (supervisor and quality)"
	case constants.ReleaseRequirementFinal:
		return "final quality"
	default:
		return string(requirement)
	}
}
