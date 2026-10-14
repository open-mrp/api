package repository

import (
	"context"
	gosql "database/sql"
	"encoding/json"
	"errors"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var batchReleaseRepoTracer = tracing.GetTracer("core-service.batch_release_repository")

// ---------------------------------------------------------------------------
// Batch signatures
// ---------------------------------------------------------------------------

type batchSignatureRepoImpl struct {
	queries *sqlc.Queries
}

func NewBatchSignatureRepo(queries *sqlc.Queries) domain.BatchSignatureRepo {
	return &batchSignatureRepoImpl{queries: queries}
}

func (r *batchSignatureRepoImpl) Create(ctx context.Context, signature *domain.BatchSignature) (*domain.BatchSignature, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.batch_signature.create")
	defer span.End()

	var checklist json.RawMessage
	if len(signature.Checklist) > 0 {
		raw, err := json.Marshal(signature.Checklist)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue encoding the signature checklist."))
		}
		checklist = raw
	}

	err := r.queries.InsertBatchSignature(ctx, sqlc.InsertBatchSignatureParams{
		ID:                  signature.ID,
		AccountID:           signature.AccountID,
		BatchID:             signature.BatchID,
		ProductionStepID:    signature.ProductionStepID,
		ProductionRunID:     db.NullStringPtr(signature.ProductionRunID),
		TypeCode:            string(signature.Type),
		DecisionCode:        string(signature.Decision),
		CapacityCode:        string(signature.Capacity),
		SignerAccountUserID: signature.SignerAccountUserID,
		SignerName:          signature.SignerName,
		SignerRoleName:      db.NullStringPtr(signature.SignerRoleName),
		Checklist:           db.NullableRawMessage(checklist),
		Note:                db.NullStringPtr(signature.Note),
		ActorID:             signature.ActorID,
		ActorType:           signature.ActorType,
		LotID:               db.NullStringPtr(signature.LotID),
		LotNumber:           db.NullStringPtr(signature.LotNumber),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, signature.AccountID, signature.ID)
}

func (r *batchSignatureRepoImpl) Get(ctx context.Context, accountID, id string) (*domain.BatchSignature, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.batch_signature.get")
	defer span.End()

	row, err := r.queries.GetBatchSignature(ctx, sqlc.GetBatchSignatureParams{ID: id, AccountID: accountID})
	if errors.Is(err, gosql.ErrNoRows) {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Batch signature not found."))
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	signature, apiErr := mapBatchSignature(row)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return signature, nil
}

func (r *batchSignatureRepoImpl) ListForBatch(ctx context.Context, accountID, batchID string) ([]*domain.BatchSignature, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.batch_signature.list_for_batch")
	defer span.End()

	rows, err := r.queries.ListBatchSignaturesForBatch(ctx, sqlc.ListBatchSignaturesForBatchParams{AccountID: accountID, BatchID: batchID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	signatures, apiErr := mapBatchSignatures(rows)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return signatures, nil
}

func (r *batchSignatureRepoImpl) ListForBatches(ctx context.Context, accountID string, batchIDs []string) ([]*domain.BatchSignature, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.batch_signature.list_for_batches")
	defer span.End()

	if len(batchIDs) == 0 {
		return nil, nil
	}
	rows, err := r.queries.ListBatchSignaturesForBatches(ctx, sqlc.ListBatchSignaturesForBatchesParams{AccountID: accountID, BatchIds: batchIDs})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	signatures, apiErr := mapBatchSignatures(rows)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return signatures, nil
}

func (r *batchSignatureRepoImpl) ListRunStepReleases(ctx context.Context, accountID, productionRunID, productionStepID string) ([]*domain.BatchSignature, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.batch_signature.list_run_step_releases")
	defer span.End()

	rows, err := r.queries.ListRunStepReleaseSignatures(ctx, sqlc.ListRunStepReleaseSignaturesParams{
		AccountID:        accountID,
		ProductionRunID:  gosql.NullString{String: productionRunID, Valid: true},
		ProductionStepID: productionStepID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	signatures, apiErr := mapBatchSignatures(rows)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return signatures, nil
}

func (r *batchSignatureRepoImpl) ListSignerCandidates(ctx context.Context, accountID string) ([]domain.SignerCandidate, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.batch_signature.list_signer_candidates")
	defer span.End()

	rows, err := r.queries.ListEligibleSigners(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	candidates := make([]domain.SignerCandidate, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, domain.SignerCandidate{
			AccountUserID: row.AccountUserID,
			Name:          row.Name.String,
			RoleName:      row.RoleName,
			DepartmentID:  db.StringFromNullString(row.DepartmentID),
			IsAdmin:       row.RoleTypeCode == string(constants.RoleTypeAdmin),
			CanSupervise:  row.CanSupervise,
			CanInspect:    row.CanInspect,
		})
	}
	return candidates, nil
}

func mapBatchSignatures(rows []sqlc.BatchSignature) ([]*domain.BatchSignature, *apierror.APIError) {
	signatures := make([]*domain.BatchSignature, 0, len(rows))
	for _, row := range rows {
		signature, apiErr := mapBatchSignature(row)
		if apiErr != nil {
			return nil, apiErr
		}
		signatures = append(signatures, signature)
	}
	return signatures, nil
}

func mapBatchSignature(row sqlc.BatchSignature) (*domain.BatchSignature, *apierror.APIError) {
	signature := &domain.BatchSignature{
		ID:                  row.ID,
		AccountID:           row.AccountID,
		BatchID:             row.BatchID,
		ProductionStepID:    row.ProductionStepID,
		ProductionRunID:     db.StringFromNullString(row.ProductionRunID),
		Type:                constants.BatchSignatureType(row.TypeCode),
		Decision:            constants.BatchSignatureDecision(row.DecisionCode),
		Capacity:            constants.SignatureCapacity(row.CapacityCode),
		SignerAccountUserID: row.SignerAccountUserID,
		SignerName:          row.SignerName,
		SignerRoleName:      db.StringFromNullString(row.SignerRoleName),
		Note:                db.StringFromNullString(row.Note),
		ActorID:             row.ActorID,
		ActorType:           row.ActorType,
		LotID:               db.StringFromNullString(row.LotID),
		LotNumber:           db.StringFromNullString(row.LotNumber),
		SignedAt:            row.SignedAt,
	}
	if len(row.Checklist) > 0 {
		if err := json.Unmarshal(row.Checklist, &signature.Checklist); err != nil {
			return nil, apierror.NewInternalError(err, "Issue decoding the signature checklist.")
		}
	}
	return signature, nil
}

// ---------------------------------------------------------------------------
// Operator check-ins
// ---------------------------------------------------------------------------

type operatorCheckInRepoImpl struct {
	queries *sqlc.Queries
}

func NewOperatorCheckInRepo(queries *sqlc.Queries) domain.OperatorCheckInRepo {
	return &operatorCheckInRepoImpl{queries: queries}
}

func (r *operatorCheckInRepoImpl) Create(ctx context.Context, checkIn *domain.OperatorCheckIn) (*domain.OperatorCheckIn, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.operator_check_in.create")
	defer span.End()

	err := r.queries.InsertOperatorCheckIn(ctx, sqlc.InsertOperatorCheckInParams{
		ID:                checkIn.ID,
		AccountID:         checkIn.AccountID,
		ScanningStationID: checkIn.ScanningStationID,
		AccountUserID:     checkIn.AccountUserID,
		OperatorName:      checkIn.OperatorName,
		ExpiresAt:         checkIn.ExpiresAt,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return r.Get(ctx, checkIn.AccountID, checkIn.ID)
}

func (r *operatorCheckInRepoImpl) Get(ctx context.Context, accountID, id string) (*domain.OperatorCheckIn, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.operator_check_in.get")
	defer span.End()

	row, err := r.queries.GetOperatorCheckIn(ctx, sqlc.GetOperatorCheckInParams{ID: id, AccountID: accountID})
	if errors.Is(err, gosql.ErrNoRows) {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Operator check-in not found."))
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapOperatorCheckIn(row), nil
}

func (r *operatorCheckInRepoImpl) GetLatestForStation(ctx context.Context, accountID, scanningStationID string) (*domain.OperatorCheckIn, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.operator_check_in.get_latest_for_station")
	defer span.End()

	row, err := r.queries.GetActiveOperatorCheckIn(ctx, sqlc.GetActiveOperatorCheckInParams{AccountID: accountID, ScanningStationID: scanningStationID})
	if errors.Is(err, gosql.ErrNoRows) {
		return nil, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapOperatorCheckIn(row), nil
}

func (r *operatorCheckInRepoImpl) EndOpenForStation(ctx context.Context, accountID, scanningStationID string) *apierror.APIError {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.operator_check_in.end_open_for_station")
	defer span.End()

	err := r.queries.EndOpenOperatorCheckIns(ctx, sqlc.EndOpenOperatorCheckInsParams{AccountID: accountID, ScanningStationID: scanningStationID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *operatorCheckInRepoImpl) CheckOut(ctx context.Context, accountID, id string) (bool, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.operator_check_in.check_out")
	defer span.End()

	affected, err := r.queries.CheckOutOperatorCheckIn(ctx, sqlc.CheckOutOperatorCheckInParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return affected > 0, nil
}

func mapOperatorCheckIn(row sqlc.OperatorCheckIn) *domain.OperatorCheckIn {
	return &domain.OperatorCheckIn{
		ID:                row.ID,
		AccountID:         row.AccountID,
		ScanningStationID: row.ScanningStationID,
		AccountUserID:     row.AccountUserID,
		OperatorName:      row.OperatorName,
		CheckedInAt:       row.CheckedInAt,
		ExpiresAt:         row.ExpiresAt,
		CheckedOutAt:      db.TimeFromNullTime(row.CheckedOutAt),
	}
}

// ---------------------------------------------------------------------------
// Department release policies
// ---------------------------------------------------------------------------

type releasePolicyRepoImpl struct {
	queries *sqlc.Queries
}

func NewReleasePolicyRepo(queries *sqlc.Queries) domain.ReleasePolicyRepo {
	return &releasePolicyRepoImpl{queries: queries}
}

func (r *releasePolicyRepoImpl) Get(ctx context.Context, accountID, departmentID string) (*domain.ReleasePolicy, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.release_policy.get")
	defer span.End()

	row, err := r.queries.GetDepartmentReleasePolicy(ctx, sqlc.GetDepartmentReleasePolicyParams{ID: departmentID, AccountID: accountID})
	if errors.Is(err, gosql.ErrNoRows) {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Department not found."))
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	policy, apiErr := mapReleasePolicy(sqlc.ListDepartmentReleasePoliciesRow(row))
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return policy, nil
}

func (r *releasePolicyRepoImpl) ListByDepartmentIDs(ctx context.Context, accountID string, departmentIDs []string) (map[string]*domain.ReleasePolicy, *apierror.APIError) {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.release_policy.list_by_department_ids")
	defer span.End()

	policies := make(map[string]*domain.ReleasePolicy, len(departmentIDs))
	if len(departmentIDs) == 0 {
		return policies, nil
	}
	rows, err := r.queries.ListDepartmentReleasePolicies(ctx, sqlc.ListDepartmentReleasePoliciesParams{AccountID: accountID, Ids: departmentIDs})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, row := range rows {
		policy, apiErr := mapReleasePolicy(row)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		policies[policy.DepartmentID] = policy
	}
	return policies, nil
}

func (r *releasePolicyRepoImpl) Update(ctx context.Context, accountID string, params domain.UpdateReleasePolicyParams) *apierror.APIError {
	ctx, span := batchReleaseRepoTracer.Start(ctx, "repository.release_policy.update")
	defer span.End()

	var checklist json.RawMessage
	if len(params.Checklist) > 0 {
		raw, err := json.Marshal(params.Checklist)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue encoding the release checklist."))
		}
		checklist = raw
	}

	err := r.queries.UpdateDepartmentReleasePolicy(ctx, sqlc.UpdateDepartmentReleasePolicyParams{
		ID:                         params.DepartmentID,
		AccountID:                  accountID,
		ReleaseRequirementCode:     string(params.Requirement),
		ReleaseEnforcementCode:     string(params.Enforcement),
		ReleaseOperatorSigningCode: string(params.OperatorSigning),
		ReleaseRationale:           db.NullStringPtr(params.Rationale),
		ReleaseChecklist:           db.NullableRawMessage(checklist),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func mapReleasePolicy(row sqlc.ListDepartmentReleasePoliciesRow) (*domain.ReleasePolicy, *apierror.APIError) {
	policy := &domain.ReleasePolicy{
		DepartmentID:    row.ID,
		DepartmentName:  row.Name,
		AccountID:       row.AccountID,
		Requirement:     constants.ReleaseRequirement(row.ReleaseRequirementCode),
		Enforcement:     constants.ReleaseEnforcement(row.ReleaseEnforcementCode),
		OperatorSigning: constants.ReleaseOperatorSigning(row.ReleaseOperatorSigningCode),
		Rationale:       db.StringFromNullString(row.ReleaseRationale),
		UpdatedAt:       row.UpdatedAt,
	}
	if len(row.ReleaseChecklist) > 0 {
		if err := json.Unmarshal(row.ReleaseChecklist, &policy.Checklist); err != nil {
			return nil, apierror.NewInternalError(err, "Issue decoding the release checklist.")
		}
	}
	return policy, nil
}
