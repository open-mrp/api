package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var batchReleaseSvcTracer = tracing.GetTracer("core-service.batch_release_service")

const (
	// operatorCheckInLifetime outlasts a long shift, so an operator who forgets to check out stops
	// being credited with scans by the next one.
	operatorCheckInLifetime = 12 * time.Hour

	maxReleaseChecklistPrompts = 20
	maxReleaseChecklistPrompt  = 255
	maxSignatureNote           = 2000
)

type batchReleaseSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	authClient      domain.CoreAuthClient
}

type BatchReleaseSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// AuthClient (required) checks signing PINs.
	AuthClient domain.CoreAuthClient
}

func (c *BatchReleaseSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("batch release service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("batch release service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("batch release service: tx manager is required")
	}
	if c.AuthClient == nil {
		return fmt.Errorf("batch release service: auth client is required")
	}
	return nil
}

func NewBatchReleaseSvc(config *BatchReleaseSvcConfig) domain.BatchReleaseSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &batchReleaseSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		authClient:      config.AuthClient,
	}
}

func (s *batchReleaseSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *batchReleaseSvcImpl) withTx(ctx context.Context, fn func(context.Context, *batchReleaseSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		return fn(txCtx, &batchReleaseSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
			authClient:      s.authClient,
		})
	})
}

// releaseActor resolves the internal caller and checks one permission.
func releaseActor(ctx context.Context, domainName types.PermissionDomain, action types.Action) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := identity.CheckHasPermission(domainName, action); apiErr != nil {
		return nil, apiErr
	}
	return identity, nil
}

// runIdempotent runs a write once per idempotency key, replaying what the first run returned.
func runIdempotent[T any](ctx context.Context, meds domain.Mediators, identity *types.Identity, run func(key *domain.IdempotencyKey) (*T, *apierror.APIError)) (*T, *apierror.APIError) {
	key, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}
	switch domain.RecoveryPoint(key.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[T](ctx, key.ResponseCode, key.ResponseBody)
		if err != nil {
			return nil, apierror.NewInternalError(err, "Issue unmarshalling cached response.")
		}
		return cached.Data, cached.Error
	case domain.RecoveryPointStarted:
		result, apiErr := run(key)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, key.TypeID, apiErr)
		}
		return result, nil
	default:
		return nil, apierror.NewInvariantViolationError("Unexpected recovery point: " + key.RecoveryPoint)
	}
}

// ---------------------------------------------------------------------------
// Release policies
// ---------------------------------------------------------------------------

func (s *batchReleaseSvcImpl) GetReleasePolicy(ctx context.Context, departmentID string) (*domain.ReleasePolicy, *apierror.APIError) {
	ctx, span := batchReleaseSvcTracer.Start(ctx, "service.batch_release.get_release_policy")
	defer span.End()

	identity, apiErr := releaseActor(ctx, types.PermissionDomainDepartments, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	policy, apiErr := s.repos.NewReleasePolicyRepo().Get(ctx, identity.Target.AccountID, departmentID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return policy, nil
}

// UpdateReleasePolicy replaces the sign-off a department requires.
//
//  1. Validate the requirement, enforcement and checklist prompts.
//  2. Under an idempotency key, write the policy and publish its audit change on the step, so who set
//     the requirement, when, and the rationale behind it stay on record.
func (s *batchReleaseSvcImpl) UpdateReleasePolicy(ctx context.Context, params domain.UpdateReleasePolicyParams) (*domain.ReleasePolicy, *apierror.APIError) {
	ctx, span := batchReleaseSvcTracer.Start(ctx, "service.batch_release.update_release_policy")
	defer span.End()

	identity, apiErr := releaseActor(ctx, types.PermissionDomainDepartments, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	if !params.Requirement.IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unknown release requirement.", "requirement"))
	}
	if !params.Enforcement.IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unknown release enforcement.", "enforcement"))
	}
	if params.OperatorSigning == "" {
		params.OperatorSigning = constants.ReleaseOperatorSigningAllowed
	}
	if !params.OperatorSigning.IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unknown operator signing rule.", "operator_signing"))
	}
	checklist, apiErr := normalizeChecklistPrompts(params.Checklist)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.Checklist = checklist
	if params.Rationale != nil {
		trimmed := strings.TrimSpace(*params.Rationale)
		params.Rationale = &trimmed
		if trimmed == "" {
			params.Rationale = nil
		}
	}

	result, apiErr := runIdempotent(ctx, s.mediators(), identity, func(key *domain.IdempotencyKey) (*domain.ReleasePolicy, *apierror.APIError) {
		var result *domain.ReleasePolicy
		apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *batchReleaseSvcImpl) *apierror.APIError {
			policyRepo := txSvc.repos.NewReleasePolicyRepo()
			existing, apiErr := policyRepo.Get(txCtx, accountID, params.DepartmentID)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := policyRepo.Update(txCtx, accountID, params); apiErr != nil {
				return apiErr
			}
			updated, apiErr := policyRepo.Get(txCtx, accountID, params.DepartmentID)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeReleasePolicy,
				ResourceID:   params.DepartmentID,
				Changes:      audit.ComputeChanges(existing, updated),
			}); apiErr != nil {
				return apiErr
			}
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, key.TypeID, result)
		})
		return result, apiErr
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

func normalizeChecklistPrompts(prompts []string) ([]string, *apierror.APIError) {
	if len(prompts) > maxReleaseChecklistPrompts {
		return nil, apierror.NewValidationErrorWithParam(fmt.Sprintf("A checklist can have at most %d prompts.", maxReleaseChecklistPrompts), "checklist")
	}
	seen := make(map[string]bool, len(prompts))
	normalized := make([]string, 0, len(prompts))
	for _, prompt := range prompts {
		prompt = strings.TrimSpace(prompt)
		switch {
		case prompt == "":
			return nil, apierror.NewValidationErrorWithParam("Checklist prompts can't be blank.", "checklist")
		case len(prompt) > maxReleaseChecklistPrompt:
			return nil, apierror.NewValidationErrorWithParam(fmt.Sprintf("Checklist prompts can be at most %d characters.", maxReleaseChecklistPrompt), "checklist")
		case seen[prompt]:
			return nil, apierror.NewValidationErrorWithParam("Checklist prompts must be different from each other.", "checklist")
		}
		seen[prompt] = true
		normalized = append(normalized, prompt)
	}
	return normalized, nil
}

// ---------------------------------------------------------------------------
// Signatures
// ---------------------------------------------------------------------------

func (s *batchReleaseSvcImpl) ListBatchSignatures(ctx context.Context, batchID string) ([]*domain.BatchSignature, *apierror.APIError) {
	ctx, span := batchReleaseSvcTracer.Start(ctx, "service.batch_release.list_batch_signatures")
	defer span.End()

	identity, apiErr := releaseActor(ctx, types.PermissionDomainBatches, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	signatures, apiErr := s.repos.NewBatchSignatureRepo().ListForBatch(ctx, identity.Target.AccountID, batchID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return signatures, nil
}

// SignBatch records an electronic signature on a batch at the step it is at.
//
//  1. Validate the request and work out what the signature is (release or correction) and the
//     capacity it is signed in from the step's requirement.
//  2. Under an idempotency key, check the signer's PIN with auth-service before any transaction, then
//     check they may sign in that capacity: quality needs quality_releases, supervisor needs
//     batch_releases and, for a supervisor with a department, the step's department.
//  3. An approving final release also needs every upstream sign-off to exist.
//  4. Write the signature and its audit event, and cache the response, in one transaction.
func (s *batchReleaseSvcImpl) SignBatch(ctx context.Context, params domain.SignBatchParams) (*domain.BatchSignature, *apierror.APIError) {
	ctx, span := batchReleaseSvcTracer.Start(ctx, "service.batch_release.sign_batch")
	defer span.End()

	identity, apiErr := releaseActor(ctx, types.PermissionDomainBatches, types.ActionCreate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	if !params.Decision.IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unknown decision.", "decision"))
	}
	if params.Type != nil && *params.Type != constants.BatchSignatureTypeCorrection {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Only a correction is asked for by type; a release's type follows from the step.", "type"))
	}
	if params.Capacity != nil && !params.Capacity.IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Unknown capacity.", "capacity"))
	}
	isCorrection := params.Type != nil
	if params.Note != nil {
		trimmed := strings.TrimSpace(*params.Note)
		params.Note = &trimmed
		if trimmed == "" {
			params.Note = nil
		} else if len(trimmed) > maxSignatureNote {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam(fmt.Sprintf("A note can be at most %d characters.", maxSignatureNote), "note"))
		}
	}
	if params.Note == nil && (isCorrection || params.Decision == constants.BatchSignatureDecisionRejected) {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Say why in the note.", "note"))
	}

	batch, apiErr := s.repos.NewBatchRepo().Find(ctx, accountID, params.BatchID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Batch not found."))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	if batch.ScannedAt == nil || batch.ProductionStep == nil || batch.DepartmentID == nil {
		return nil, tracing.Trace(span, apierror.NewResourceConflictError("This batch hasn't been scanned yet, so there is nothing to sign off."))
	}
	policy, apiErr := s.repos.NewReleasePolicyRepo().Get(ctx, accountID, *batch.DepartmentID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if policy.Requirement == constants.ReleaseRequirementNone {
		return nil, tracing.Trace(span, apierror.NewResourceConflictError(fmt.Sprintf("%s doesn't require a sign-off.", policy.DepartmentName)))
	}

	signatureType := constants.BatchSignatureTypeStepRelease
	switch {
	case isCorrection:
		signatureType = constants.BatchSignatureTypeCorrection
	case policy.Requirement == constants.ReleaseRequirementFinal:
		signatureType = constants.BatchSignatureTypeFinalRelease
	}
	capacity, apiErr := signatureCapacity(policy.Requirement, params.Capacity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var checklist []domain.ChecklistResponse
	if !isCorrection {
		checklist, apiErr = matchChecklist(policy.Checklist, params.Checklist)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if len(params.Checklist) > 0 {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("A correction has no checklist.", "checklist"))
	}

	signatureID, apiErr := id.GenID(id.BatchSignatureIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, apiErr := runIdempotent(ctx, s.mediators(), identity, func(key *domain.IdempotencyKey) (*domain.BatchSignature, *apierror.APIError) {
		signer, apiErr := s.authClient.VerifySigningPin(ctx, params.SignerAccountUserID, params.PIN)
		if apiErr != nil {
			return nil, apiErr
		}
		if apiErr := checkSignerMay(signer, capacity, policy); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := checkOperatorMaySign(signer, batch, policy); apiErr != nil {
			return nil, apiErr
		}
		if signatureType == constants.BatchSignatureTypeFinalRelease && params.Decision == constants.BatchSignatureDecisionApproved {
			if apiErr := s.mediators().BatchRelease.CheckLineageReleased(ctx, accountID, batch.ID); apiErr != nil {
				return nil, apiErr
			}
		}

		var runID *string
		if batch.ProductionRun != nil {
			runID = &batch.ProductionRun.ID
		}
		var roleName *string
		if signer.RoleName != "" {
			roleName = &signer.RoleName
		}
		actorID, actorType := "", string(identity.Type)
		if identity.Actor != nil {
			actorID = identity.Actor.ID
		}

		assignsLot := signatureType == constants.BatchSignatureTypeFinalRelease && params.Decision == constants.BatchSignatureDecisionApproved
		var lotZone *time.Location
		if assignsLot {
			lotZone = s.releaseLotZone(ctx, accountID)
		}

		var result *domain.BatchSignature
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchReleaseSvcImpl) *apierror.APIError {
			var lot *domain.ReleaseLot
			if assignsLot {
				assigned, apiErr := txSvc.assignReleaseLot(txCtx, accountID, batch, lotZone)
				if apiErr != nil {
					return apiErr
				}
				lot = assigned
			}
			var lotID, lotNumber *string
			if lot != nil {
				lotID, lotNumber = &lot.ID, &lot.LotNumber
			}

			created, apiErr := txSvc.repos.NewBatchSignatureRepo().Create(txCtx, &domain.BatchSignature{
				ID:                  signatureID,
				AccountID:           accountID,
				BatchID:             batch.ID,
				ProductionStepID:    batch.ProductionStep.ID,
				ProductionRunID:     runID,
				Type:                signatureType,
				Decision:            params.Decision,
				Capacity:            capacity,
				SignerAccountUserID: signer.AccountUserID,
				SignerName:          signer.Name,
				SignerRoleName:      roleName,
				Checklist:           checklist,
				Note:                params.Note,
				ActorID:             actorID,
				ActorType:           actorType,
				LotID:               lotID,
				LotNumber:           lotNumber,
			})
			if apiErr != nil {
				return apiErr
			}
			result = created

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeBatchSignature,
				ResourceID:   created.ID,
				Changes:      audit.ComputeChanges(nil, created),
			}); apiErr != nil {
				return apiErr
			}
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, key.TypeID, result)
		})
		return result, apiErr
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

// assignReleaseLot gives a finally released batch its finished-goods lot: the account's next sequence
// number and the Julian date of the release (two-digit year and day of the year) in the plant's zone,
// such as 1234-26286. A batch released again after a withdrawal keeps the lot it was given first.
func (s *batchReleaseSvcImpl) assignReleaseLot(ctx context.Context, accountID string, batch *domain.Batch, zone *time.Location) (*domain.ReleaseLot, *apierror.APIError) {
	batchRepo := s.repos.NewBatchRepo()
	existing, apiErr := batchRepo.GetReleaseLot(ctx, accountID, batch.ID)
	if apiErr != nil || existing != nil {
		return existing, apiErr
	}

	sequence, apiErr := batchRepo.AllocateReleaseLotSequence(ctx, accountID)
	if apiErr != nil {
		return nil, apiErr
	}
	lotID, apiErr := id.GenID(id.LotIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	releasedAt := time.Now()
	lot := domain.ReleaseLot{ID: lotID, LotNumber: releaseLotNumber(sequence, releasedAt.In(zone))}
	if apiErr := batchRepo.CreateReleaseLot(ctx, accountID, batch.ID, batch.Item.ID, lot, releasedAt); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := audit.NewPublisher().Publish(ctx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionCreate,
		ResourceType: constants.ObjectTypeLot,
		ResourceID:   lot.ID,
		Changes: []audit.FieldChange{
			audit.NewFieldChange("lot_number", nil, lot.LotNumber),
			audit.NewFieldChange("item_id", nil, batch.Item.ID),
			audit.NewFieldChange("batch_id", nil, batch.ID),
		},
	}); apiErr != nil {
		return nil, apiErr
	}
	return &lot, nil
}

// releaseLotNumber formats a release lot as its sequence number and the Julian date it was released on.
func releaseLotNumber(sequence int64, releasedAt time.Time) string {
	return fmt.Sprintf("%d-%02d%03d", sequence, releasedAt.Year()%100, releasedAt.YearDay())
}

// releaseLotZone is the zone a release's date is read in: the plant's shift zone when the account has
// set one, otherwise UTC.
func (s *batchReleaseSvcImpl) releaseLotZone(ctx context.Context, accountID string) *time.Location {
	settings, apiErr := s.repos.NewProductionScheduleRepo().GetSettings(ctx, accountID)
	if apiErr != nil || settings == nil || settings.ShiftTimezone == nil {
		return time.UTC
	}
	zone, err := time.LoadLocation(*settings.ShiftTimezone)
	if err != nil {
		return time.UTC
	}
	return zone
}

// checkOperatorMaySign refuses the operator who scanned a batch at a step that wants someone else to sign it.
func checkOperatorMaySign(signer *domain.Signer, batch *domain.Batch, policy *domain.ReleasePolicy) *apierror.APIError {
	if policy.OperatorSigning != constants.ReleaseOperatorSigningForbidden || batch.Operator == nil {
		return nil
	}
	if batch.Operator.AccountUserID == signer.AccountUserID {
		return apierror.NewAuthorizationError(fmt.Sprintf("%s scanned this batch, and this step needs someone else to sign it off.", signer.Name))
	}
	return nil
}

// signatureCapacity is the authority a signature is made in. A first-article step needs one of each, so
// the request says which; elsewhere it follows from the requirement and a request may only repeat it.
func signatureCapacity(requirement constants.ReleaseRequirement, requested *constants.SignatureCapacity) (constants.SignatureCapacity, *apierror.APIError) {
	var required constants.SignatureCapacity
	switch requirement {
	case constants.ReleaseRequirementSupervisor:
		required = constants.SignatureCapacitySupervisor
	case constants.ReleaseRequirementQuality, constants.ReleaseRequirementFinal:
		required = constants.SignatureCapacityQuality
	case constants.ReleaseRequirementFirstArticle:
		if requested == nil {
			return "", apierror.NewValidationErrorWithParam("A first-article sign-off needs a supervisor and a quality signature; say which this is.", "capacity")
		}
		return *requested, nil
	}
	if requested != nil && *requested != required {
		return "", apierror.NewValidationErrorWithParam(fmt.Sprintf("This step is signed off in the %s capacity.", required), "capacity")
	}
	return required, nil
}

// checkSignerMay refuses a signer without the authority the capacity needs.
func checkSignerMay(signer *domain.Signer, capacity constants.SignatureCapacity, policy *domain.ReleasePolicy) *apierror.APIError {
	if capacity == constants.SignatureCapacityQuality {
		if !signer.HasPermission(string(types.PermissionDomainQualityReleases), string(types.ActionCreate)) {
			return apierror.NewAuthorizationError(fmt.Sprintf("%s can't sign quality releases.", signer.Name))
		}
		return nil
	}
	if !signer.HasPermission(string(types.PermissionDomainBatchReleases), string(types.ActionCreate)) {
		return apierror.NewAuthorizationError(fmt.Sprintf("%s can't sign batch releases.", signer.Name))
	}
	if !signer.IsAdmin && signer.DepartmentID != nil && *signer.DepartmentID != policy.DepartmentID {
		return apierror.NewAuthorizationError(fmt.Sprintf("%s supervises another department.", signer.Name))
	}
	return nil
}

// matchChecklist pairs the signer's answers with the step's prompts. Every prompt must be answered, and
// an answer to a prompt the step doesn't have means the signer saw an out-of-date checklist.
func matchChecklist(prompts []string, answers []domain.ChecklistResponse) ([]domain.ChecklistResponse, *apierror.APIError) {
	byPrompt := make(map[string]constants.ChecklistAnswer, len(answers))
	for _, answer := range answers {
		if !answer.Answer.IsValid() {
			return nil, apierror.NewValidationErrorWithParam("Unknown checklist answer.", "checklist")
		}
		byPrompt[strings.TrimSpace(answer.Prompt)] = answer.Answer
	}
	if len(byPrompt) != len(answers) {
		return nil, apierror.NewValidationErrorWithParam("Each checklist prompt can be answered once.", "checklist")
	}
	matched := make([]domain.ChecklistResponse, 0, len(prompts))
	for _, prompt := range prompts {
		answer, ok := byPrompt[prompt]
		if !ok {
			return nil, apierror.NewValidationErrorWithParam(fmt.Sprintf("Answer the checklist prompt %q.", prompt), "checklist")
		}
		matched = append(matched, domain.ChecklistResponse{Prompt: prompt, Answer: answer})
		delete(byPrompt, prompt)
	}
	if len(byPrompt) > 0 {
		return nil, apierror.NewValidationErrorWithParam("The checklist has changed; reload it and answer again.", "checklist")
	}
	return matched, nil
}

// ListEligibleSigners returns who may sign a batch off in its department, and in which capacities,
// leaving out the operator who scanned it when the department needs someone else to sign.
func (s *batchReleaseSvcImpl) ListEligibleSigners(ctx context.Context, batchID string) ([]domain.EligibleSigner, *apierror.APIError) {
	ctx, span := batchReleaseSvcTracer.Start(ctx, "service.batch_release.list_eligible_signers")
	defer span.End()

	identity, apiErr := releaseActor(ctx, types.PermissionDomainBatches, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	batch, apiErr := s.repos.NewBatchRepo().Find(ctx, accountID, batchID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Batch not found."))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	if batch.DepartmentID == nil {
		return []domain.EligibleSigner{}, nil
	}
	policy, apiErr := s.repos.NewReleasePolicyRepo().Get(ctx, accountID, *batch.DepartmentID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var excluded string
	if policy.OperatorSigning == constants.ReleaseOperatorSigningForbidden && batch.Operator != nil {
		excluded = batch.Operator.AccountUserID
	}

	candidates, apiErr := s.repos.NewBatchSignatureRepo().ListSignerCandidates(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	wanted := capacitiesFor(policy.Requirement)
	signers := make([]domain.EligibleSigner, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.AccountUserID == excluded {
			continue
		}
		var capacities []constants.SignatureCapacity
		for _, capacity := range wanted {
			if candidateMay(candidate, capacity, policy) {
				capacities = append(capacities, capacity)
			}
		}
		if len(capacities) > 0 {
			signers = append(signers, domain.EligibleSigner{
				AccountUserID: candidate.AccountUserID,
				Name:          candidate.Name,
				RoleName:      candidate.RoleName,
				Capacities:    capacities,
			})
		}
	}
	return signers, nil
}

// capacitiesFor lists the capacities a step's batches are signed off in.
func capacitiesFor(requirement constants.ReleaseRequirement) []constants.SignatureCapacity {
	switch requirement {
	case constants.ReleaseRequirementSupervisor:
		return []constants.SignatureCapacity{constants.SignatureCapacitySupervisor}
	case constants.ReleaseRequirementQuality, constants.ReleaseRequirementFinal:
		return []constants.SignatureCapacity{constants.SignatureCapacityQuality}
	case constants.ReleaseRequirementFirstArticle:
		return []constants.SignatureCapacity{constants.SignatureCapacitySupervisor, constants.SignatureCapacityQuality}
	default:
		return nil
	}
}

// candidateMay applies checkSignerMay's rules to a candidate read from the roles.
func candidateMay(candidate domain.SignerCandidate, capacity constants.SignatureCapacity, policy *domain.ReleasePolicy) bool {
	permissions := map[string]bool{}
	if candidate.CanSupervise {
		permissions[string(types.PermissionDomainBatchReleases)+":"+string(types.ActionCreate)] = true
	}
	if candidate.CanInspect {
		permissions[string(types.PermissionDomainQualityReleases)+":"+string(types.ActionCreate)] = true
	}
	signer := &domain.Signer{
		AccountUserID: candidate.AccountUserID,
		Name:          candidate.Name,
		IsAdmin:       candidate.IsAdmin,
		DepartmentID:  candidate.DepartmentID,
		Permissions:   permissions,
	}
	return checkSignerMay(signer, capacity, policy) == nil
}

// ---------------------------------------------------------------------------
// Operator check-ins
// ---------------------------------------------------------------------------

// CheckInOperator identifies who is working at a station, by their signing PIN.
//
//  1. Under an idempotency key, check the station is in the account and the operator's PIN, before any transaction.
//  2. End whoever was checked in at the station and check the operator in until they check out or the check-in expires.
//  3. Publish the audit event and cache the response in the same transaction.
func (s *batchReleaseSvcImpl) CheckInOperator(ctx context.Context, params domain.CheckInOperatorParams) (*domain.OperatorCheckIn, *apierror.APIError) {
	ctx, span := batchReleaseSvcTracer.Start(ctx, "service.batch_release.check_in_operator")
	defer span.End()

	identity, apiErr := releaseActor(ctx, types.PermissionDomainBatches, types.ActionCreate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	inAccount, apiErr := s.repos.NewScanningStationQueryRepo().IsInAccount(ctx, accountID, params.ScanningStationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Scanning Station not found."))
	}
	checkInID, apiErr := id.GenID(id.OperatorCheckInIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, apiErr := runIdempotent(ctx, s.mediators(), identity, func(key *domain.IdempotencyKey) (*domain.OperatorCheckIn, *apierror.APIError) {
		operator, apiErr := s.authClient.VerifySigningPin(ctx, params.AccountUserID, params.PIN)
		if apiErr != nil {
			return nil, apiErr
		}

		var result *domain.OperatorCheckIn
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchReleaseSvcImpl) *apierror.APIError {
			checkInRepo := txSvc.repos.NewOperatorCheckInRepo()
			if apiErr := checkInRepo.EndOpenForStation(txCtx, accountID, params.ScanningStationID); apiErr != nil {
				return apiErr
			}
			created, apiErr := checkInRepo.Create(txCtx, &domain.OperatorCheckIn{
				ID:                checkInID,
				AccountID:         accountID,
				ScanningStationID: params.ScanningStationID,
				AccountUserID:     operator.AccountUserID,
				OperatorName:      operator.Name,
				ExpiresAt:         time.Now().Add(operatorCheckInLifetime),
			})
			if apiErr != nil {
				return apiErr
			}
			result = created

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeOperatorCheckIn,
				ResourceID:   created.ID,
				Changes:      audit.ComputeChanges(nil, created),
			}); apiErr != nil {
				return apiErr
			}
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, key.TypeID, result)
		})
		return result, apiErr
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

// CheckOutOperator ends a check-in. Checking out one that already ended returns it unchanged.
func (s *batchReleaseSvcImpl) CheckOutOperator(ctx context.Context, scanningStationID, checkInID string) (*domain.OperatorCheckIn, *apierror.APIError) {
	ctx, span := batchReleaseSvcTracer.Start(ctx, "service.batch_release.check_out_operator")
	defer span.End()

	identity, apiErr := releaseActor(ctx, types.PermissionDomainBatches, types.ActionCreate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	var result *domain.OperatorCheckIn
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *batchReleaseSvcImpl) *apierror.APIError {
		checkInRepo := txSvc.repos.NewOperatorCheckInRepo()
		existing, apiErr := checkInRepo.Get(txCtx, accountID, checkInID)
		if apiErr != nil {
			return apiErr
		}
		if existing.ScanningStationID != scanningStationID {
			return apierror.NewResourceNotFoundError("Operator check-in not found.")
		}
		ended, apiErr := checkInRepo.CheckOut(txCtx, accountID, checkInID)
		if apiErr != nil {
			return apiErr
		}
		updated, apiErr := checkInRepo.Get(txCtx, accountID, checkInID)
		if apiErr != nil {
			return apiErr
		}
		result = updated
		if !ended {
			return nil
		}
		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeOperatorCheckIn,
			ResourceID:   checkInID,
			Changes:      audit.ComputeChanges(existing, updated),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

// GetActiveOperatorCheckIn returns who is checked in at a station now.
func (s *batchReleaseSvcImpl) GetActiveOperatorCheckIn(ctx context.Context, scanningStationID string) (*domain.OperatorCheckIn, *apierror.APIError) {
	ctx, span := batchReleaseSvcTracer.Start(ctx, "service.batch_release.get_active_operator_check_in")
	defer span.End()

	identity, apiErr := releaseActor(ctx, types.PermissionDomainBatches, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	checkIn, apiErr := s.repos.NewOperatorCheckInRepo().GetLatestForStation(ctx, identity.Target.AccountID, scanningStationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if checkIn == nil || checkIn.StatusAt(time.Now()) != constants.OperatorCheckInStatusActive {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Nobody is checked in at this station."))
	}
	return checkIn, nil
}
