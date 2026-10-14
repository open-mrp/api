package constants

// ReleaseRequirement is who must sign a batch off before it leaves a production step.
type ReleaseRequirement string

const (
	// ReleaseRequirementNone means batches leave the step without a sign-off.
	ReleaseRequirementNone ReleaseRequirement = "none"
	// ReleaseRequirementSupervisor means a supervisor of the step's department signs each batch off.
	ReleaseRequirementSupervisor ReleaseRequirement = "supervisor"
	// ReleaseRequirementQuality means a quality inspector signs each batch off.
	ReleaseRequirementQuality ReleaseRequirement = "quality"
	// ReleaseRequirementFirstArticle means the set-up is verified once per production run: the first batch of the run at the step needs a supervisor and a quality sign-off, which then cover every batch of that run at the step.
	ReleaseRequirementFirstArticle ReleaseRequirement = "first_article"
	// ReleaseRequirementFinal means a quality inspector releases the finished product, after every sign-off its lineage requires exists.
	ReleaseRequirementFinal ReleaseRequirement = "final"
)

func (r ReleaseRequirement) IsValid() bool {
	switch r {
	case ReleaseRequirementNone, ReleaseRequirementSupervisor, ReleaseRequirementQuality,
		ReleaseRequirementFirstArticle, ReleaseRequirementFinal:
		return true
	default:
		return false
	}
}

func (r ReleaseRequirement) EnumValues() []string {
	return []string{
		string(ReleaseRequirementNone),
		string(ReleaseRequirementSupervisor),
		string(ReleaseRequirementQuality),
		string(ReleaseRequirementFirstArticle),
		string(ReleaseRequirementFinal),
	}
}

// ReleaseEnforcement is what happens when a batch that still needs a sign-off is scanned onward.
type ReleaseEnforcement string

const (
	// ReleaseEnforcementBlock refuses the scan until the batch is signed off.
	ReleaseEnforcementBlock ReleaseEnforcement = "block"
	// ReleaseEnforcementWarn lets the scan through; the missing sign-off stays on the batch's record and stops its final release.
	ReleaseEnforcementWarn ReleaseEnforcement = "warn"
)

func (e ReleaseEnforcement) IsValid() bool {
	switch e {
	case ReleaseEnforcementBlock, ReleaseEnforcementWarn:
		return true
	default:
		return false
	}
}

func (e ReleaseEnforcement) EnumValues() []string {
	return []string{string(ReleaseEnforcementBlock), string(ReleaseEnforcementWarn)}
}

// BatchSignatureType is what a batch signature attests to.
type BatchSignatureType string

const (
	// BatchSignatureTypeStepRelease signs a batch off at a step so it may move on.
	BatchSignatureTypeStepRelease BatchSignatureType = "step_release"
	// BatchSignatureTypeFinalRelease releases the finished product.
	BatchSignatureTypeFinalRelease BatchSignatureType = "final_release"
	// BatchSignatureTypeCorrection approves correcting a scan, such as undoing it.
	BatchSignatureTypeCorrection BatchSignatureType = "correction"
)

func (t BatchSignatureType) IsValid() bool {
	switch t {
	case BatchSignatureTypeStepRelease, BatchSignatureTypeFinalRelease, BatchSignatureTypeCorrection:
		return true
	default:
		return false
	}
}

func (t BatchSignatureType) EnumValues() []string {
	return []string{
		string(BatchSignatureTypeStepRelease),
		string(BatchSignatureTypeFinalRelease),
		string(BatchSignatureTypeCorrection),
	}
}

// IsRelease reports whether the signature decides if the batch may move on.
func (t BatchSignatureType) IsRelease() bool {
	return t == BatchSignatureTypeStepRelease || t == BatchSignatureTypeFinalRelease
}

// BatchSignatureDecision is the signer's verdict.
type BatchSignatureDecision string

const (
	// BatchSignatureDecisionApproved approves the batch (or the correction).
	BatchSignatureDecisionApproved BatchSignatureDecision = "approved"
	// BatchSignatureDecisionRejected rejects it; a rejected release puts the batch on hold.
	BatchSignatureDecisionRejected BatchSignatureDecision = "rejected"
)

func (d BatchSignatureDecision) IsValid() bool {
	switch d {
	case BatchSignatureDecisionApproved, BatchSignatureDecisionRejected:
		return true
	default:
		return false
	}
}

func (d BatchSignatureDecision) EnumValues() []string {
	return []string{string(BatchSignatureDecisionApproved), string(BatchSignatureDecisionRejected)}
}

// SignatureCapacity is the authority a person signed with.
type SignatureCapacity string

const (
	// SignatureCapacitySupervisor is a supervisor signing for their own area.
	SignatureCapacitySupervisor SignatureCapacity = "supervisor"
	// SignatureCapacityQuality is a quality inspector.
	SignatureCapacityQuality SignatureCapacity = "quality"
)

func (c SignatureCapacity) IsValid() bool {
	switch c {
	case SignatureCapacitySupervisor, SignatureCapacityQuality:
		return true
	default:
		return false
	}
}

func (c SignatureCapacity) EnumValues() []string {
	return []string{string(SignatureCapacitySupervisor), string(SignatureCapacityQuality)}
}

// ChecklistAnswer is a signer's answer to one release checklist prompt.
type ChecklistAnswer string

const (
	ChecklistAnswerYes           ChecklistAnswer = "yes"
	ChecklistAnswerNo            ChecklistAnswer = "no"
	ChecklistAnswerNotApplicable ChecklistAnswer = "not_applicable"
)

func (a ChecklistAnswer) IsValid() bool {
	switch a {
	case ChecklistAnswerYes, ChecklistAnswerNo, ChecklistAnswerNotApplicable:
		return true
	default:
		return false
	}
}

func (a ChecklistAnswer) EnumValues() []string {
	return []string{string(ChecklistAnswerYes), string(ChecklistAnswerNo), string(ChecklistAnswerNotApplicable)}
}

// OperatorCheckInRequirement is whether a scanning station scans only while an operator is checked in.
type OperatorCheckInRequirement string

const (
	// OperatorCheckInOptional records the checked-in operator on scans when there is one.
	OperatorCheckInOptional OperatorCheckInRequirement = "optional"
	// OperatorCheckInRequired refuses scans while nobody is checked in.
	OperatorCheckInRequired OperatorCheckInRequirement = "required"
)

func (r OperatorCheckInRequirement) IsValid() bool {
	switch r {
	case OperatorCheckInOptional, OperatorCheckInRequired:
		return true
	default:
		return false
	}
}

func (r OperatorCheckInRequirement) EnumValues() []string {
	return []string{string(OperatorCheckInOptional), string(OperatorCheckInRequired)}
}

// OperatorCheckInStatus is whether a check-in still identifies who is working at the station.
type OperatorCheckInStatus string

const (
	// OperatorCheckInStatusActive means scans at the station are recorded against this operator.
	OperatorCheckInStatusActive OperatorCheckInStatus = "active"
	// OperatorCheckInStatusEnded means the operator checked out, someone else checked in, or it expired.
	OperatorCheckInStatusEnded OperatorCheckInStatus = "ended"
)

func (s OperatorCheckInStatus) IsValid() bool {
	switch s {
	case OperatorCheckInStatusActive, OperatorCheckInStatusEnded:
		return true
	default:
		return false
	}
}

func (s OperatorCheckInStatus) EnumValues() []string {
	return []string{string(OperatorCheckInStatusActive), string(OperatorCheckInStatusEnded)}
}

// ReleaseOperatorSigning is whether the operator who scanned a batch may sign it off.
type ReleaseOperatorSigning string

const (
	// ReleaseOperatorSigningAllowed lets the operator sign their own batch when they may sign at all.
	ReleaseOperatorSigningAllowed ReleaseOperatorSigning = "allowed"
	// ReleaseOperatorSigningForbidden needs someone other than the operator to sign.
	ReleaseOperatorSigningForbidden ReleaseOperatorSigning = "forbidden"
)

func (o ReleaseOperatorSigning) IsValid() bool {
	switch o {
	case ReleaseOperatorSigningAllowed, ReleaseOperatorSigningForbidden:
		return true
	default:
		return false
	}
}

func (o ReleaseOperatorSigning) EnumValues() []string {
	return []string{string(ReleaseOperatorSigningAllowed), string(ReleaseOperatorSigningForbidden)}
}
