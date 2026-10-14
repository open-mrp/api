package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleBatchSignatureID = "btgn_rw7eauw0xl2o"
const SampleOperatorCheckInID = "oeki_o6hkqs9ftotg"

const sampleChecklistPrompt = "Labeling and packaging correct?"

// The sign-off a department requires before a batch leaves it.
//
// A batch is in the department of the scanning station it was last scanned at, so one policy covers every production step done there. Set it from the department's risk: work where a wrong set-up makes bad product needs a sign-off, work that can't go wrong once set up may need none. Changes are kept in the audit log with who made them.
type ReleasePolicy struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=release_policy"`
	// The department the policy belongs to.
	Department *Entity `json:"department" validate:"required"`
	// Who must sign a batch off before it leaves the department.
	//
	// `none`: nobody. `supervisor`: a supervisor of the department. `quality`: a quality inspector. `first_article`: the first batch of each production run at a step needs a supervisor and a quality sign-off, which then cover every batch of that run at the step. `final`: a quality inspector releases the finished product, once every earlier sign-off its production needed exists.
	Requirement constants.ReleaseRequirement `json:"requirement" validate:"required"`
	// What happens when a batch that hasn't been signed off is scanned onward.
	//
	// `block`: the scan is refused. `warn`: the scan goes through, the missing sign-off stays on the batch's record, and the final release is refused until it is signed.
	Enforcement constants.ReleaseEnforcement `json:"enforcement" validate:"required"`
	// Whether the operator who scanned a batch may sign it off in this department.
	//
	// `allowed`: the operator signs their own batch when their role lets them sign. `forbidden`: someone else must sign it.
	OperatorSigning constants.ReleaseOperatorSigning `json:"operator_signing" validate:"required"`
	// Why the department needs, or doesn't need, a sign-off: its risk assessment.
	Rationale *string `json:"rationale"`
	// Yes/no questions the signer answers when signing a batch off in the department.
	Checklist *List[string] `json:"checklist" validate:"required"`
	// Last-updated timestamp of the department.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleReleasePolicy = &ReleasePolicy{
	Object:          constants.ObjectTypeReleasePolicy,
	Department:      NewEntity(SampleDepartmentID, constants.ObjectTypeDepartment, new("Packing"), nil),
	Requirement:     constants.ReleaseRequirementFinal,
	Enforcement:     constants.ReleaseEnforcementBlock,
	OperatorSigning: constants.ReleaseOperatorSigningForbidden,
	Rationale:       new("Last check before the product ships."),
	Checklist:       NewList([]string{sampleChecklistPrompt}, PageInfo{}),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ReleasePolicy) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleReleasePolicy)
}

// A signer's answer to one release checklist prompt.
type ChecklistResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=checklist_response"`
	// The prompt as it read when the batch was signed.
	Prompt string `json:"prompt" validate:"required"`
	// The signer's answer.
	Answer constants.ChecklistAnswer `json:"answer" validate:"required"`
}

// An electronic signature on a batch at a production step.
//
// Signatures can't be changed or deleted. A mistaken release is withdrawn by signing again with `decision` `rejected`. The signer is identified by their signing PIN, and their name and role are recorded as they were when they signed.
type BatchSignature struct {
	// Batch signature ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=batch_signature"`
	// The batch that was signed.
	Batch *BatchReference `json:"batch" validate:"required"`
	// The production step the batch was at when it was signed.
	ProductionStep *Entity `json:"production_step" validate:"required"`
	// The production run the batch belongs to.
	ProductionRun *Entity `json:"production_run"`
	// What the signature attests to.
	//
	// `step_release`: the batch may leave its step. `final_release`: the finished product is released. `correction`: a correction to the batch's scan, such as undoing it, is approved.
	Type constants.BatchSignatureType `json:"type" validate:"required"`
	// The signer's verdict.
	//
	// A `rejected` release puts the batch on hold until a later signature approves it.
	Decision constants.BatchSignatureDecision `json:"decision" validate:"required"`
	// The authority the signer signed with.
	Capacity constants.SignatureCapacity `json:"capacity" validate:"required"`
	// The person who signed.
	Signer *Entity `json:"signer" validate:"required"`
	// The signer's role when they signed.
	SignerRole *string `json:"signer_role"`
	// The step's checklist prompts and the signer's answers. Empty for a correction.
	Checklist *List[ChecklistResponse] `json:"checklist" validate:"required"`
	// Why the batch was rejected or the correction was needed, or any other remark.
	Note *string `json:"note"`
	// The finished-goods lot an approving final release assigned: the account's next sequence number and the Julian date of the release, such as `1234-26286`.
	//
	// `null` on every other signature. A batch released again after a withdrawal keeps its first lot.
	Lot *Entity `json:"lot"`
	// The credentials that submitted the signature, such as a scanning station's login.
	Actor *Actor `json:"actor" validate:"required"`
	// When the batch was signed.
	SignedAt time.Time `json:"signed_at" validate:"required"`
}

var SampleBatchSignature = &BatchSignature{
	ID:             SampleBatchSignatureID,
	Object:         constants.ObjectTypeBatchSignature,
	Batch:          &BatchReference{ID: SampleBatchID, Object: constants.ObjectTypeBatch},
	ProductionStep: NewEntity(SampleProductionStepID, constants.ObjectTypeProductionStep, new("Packaging"), nil),
	ProductionRun:  NewEntity(SampleProductionRunID, constants.ObjectTypeProductionRun, new("PR-00042"), nil),
	Type:           constants.BatchSignatureTypeFinalRelease,
	Decision:       constants.BatchSignatureDecisionApproved,
	Capacity:       constants.SignatureCapacityQuality,
	Signer:         NewEntity(SampleAccountUserID, constants.ObjectTypeAccountUser, new("Jie Yan"), nil),
	SignerRole:     new("Quality"),
	Checklist: NewList([]ChecklistResponse{{
		Object: constants.ObjectTypeChecklistResponse,
		Prompt: sampleChecklistPrompt,
		Answer: constants.ChecklistAnswerYes,
	}}, PageInfo{}),
	Lot:      NewEntity(SampleLotID, constants.ObjectTypeLot, new("1234-26286"), nil),
	Actor:    NewActor(SampleAccountUserID, constants.ActorTypeUser, new("Packing station"), nil),
	SignedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*BatchSignature) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleBatchSignature)
}

// An operator checked in at a scanning station.
//
// While a check-in is active, every batch scanned at the station records the operator. It ends when the operator checks out, when someone else checks in at the station, or 12 hours after it started.
type OperatorCheckIn struct {
	// Operator check-in ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=operator_check_in"`
	// The scanning station.
	ScanningStation *Entity `json:"scanning_station" validate:"required"`
	// The operator.
	Operator *Entity `json:"operator" validate:"required"`
	// Whether scans at the station are recorded against this operator now.
	Status constants.OperatorCheckInStatus `json:"status" validate:"required"`
	// When the operator checked in.
	CheckedInAt time.Time `json:"checked_in_at" validate:"required"`
	// When the check-in ends if the operator doesn't check out first.
	ExpiresAt time.Time `json:"expires_at" validate:"required"`
	// When the operator checked out, or someone else checked in. `null` while the check-in hasn't been ended.
	CheckedOutAt *time.Time `json:"checked_out_at"`
}

var SampleOperatorCheckIn = &OperatorCheckIn{
	ID:              SampleOperatorCheckInID,
	Object:          constants.ObjectTypeOperatorCheckIn,
	ScanningStation: NewEntity(SampleScanningStationID, constants.ObjectTypeScanningStation, new(SampleScanningStationName), nil),
	Operator:        NewEntity(SampleAccountUserID, constants.ObjectTypeAccountUser, new("Jie Yan"), nil),
	Status:          constants.OperatorCheckInStatusActive,
	CheckedInAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	ExpiresAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp).Add(12 * time.Hour),
}

func (*OperatorCheckIn) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOperatorCheckIn)
}

// A person who may sign a batch off in its department.
type EligibleSigner struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=eligible_signer"`
	// The account user.
	AccountUser *Entity `json:"account_user" validate:"required"`
	// The account user's role.
	Role string `json:"role" validate:"required"`
	// The capacities they may sign the batch off in.
	Capacities []constants.SignatureCapacity `json:"capacities" validate:"required"`
}

var SampleEligibleSigner = &EligibleSigner{
	Object:      constants.ObjectTypeEligibleSigner,
	AccountUser: NewEntity(SampleAccountUserID, constants.ObjectTypeAccountUser, new("Jie Yan"), nil),
	Role:        "Quality",
	Capacities:  []constants.SignatureCapacity{constants.SignatureCapacityQuality},
}

func (*EligibleSigner) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEligibleSigner)
}
