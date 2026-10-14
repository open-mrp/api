package domain

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
)

// ReleasePolicy is the sign-off a department requires before a batch leaves it. A batch is in the
// department of the scanning station it was last scanned at.
type ReleasePolicy struct {
	DepartmentID    string
	DepartmentName  string
	AccountID       string
	Requirement     constants.ReleaseRequirement     `audit:"requirement"`
	Enforcement     constants.ReleaseEnforcement     `audit:"enforcement"`
	OperatorSigning constants.ReleaseOperatorSigning `audit:"operator_signing"`
	Rationale       *string                          `audit:"rationale"`
	Checklist       []string                         `audit:"checklist"`
	UpdatedAt       time.Time
}

// UpdateReleasePolicyParams replaces a department's release policy as a whole.
type UpdateReleasePolicyParams struct {
	DepartmentID    string
	Requirement     constants.ReleaseRequirement
	Enforcement     constants.ReleaseEnforcement
	OperatorSigning constants.ReleaseOperatorSigning
	Rationale       *string
	Checklist       []string
}

// ChecklistResponse is a signer's answer to one release checklist prompt, with the prompt as it read
// when they signed.
type ChecklistResponse struct {
	Prompt string                    `json:"prompt"`
	Answer constants.ChecklistAnswer `json:"answer"`
}

// BatchSignature is an electronic signature on a batch at a production step. It is never changed once
// written.
type BatchSignature struct {
	ID                  string
	AccountID           string
	BatchID             string                           `audit:"batch_id"`
	ProductionStepID    string                           `audit:"production_step_id"`
	ProductionRunID     *string                          `audit:"production_run_id"`
	Type                constants.BatchSignatureType     `audit:"type"`
	Decision            constants.BatchSignatureDecision `audit:"decision"`
	Capacity            constants.SignatureCapacity      `audit:"capacity"`
	SignerAccountUserID string                           `audit:"signer_account_user_id"`
	SignerName          string                           `audit:"signer_name"`
	SignerRoleName      *string                          `audit:"signer_role_name"`
	Checklist           []ChecklistResponse              `audit:"checklist"`
	Note                *string                          `audit:"note"`
	ActorID             string
	ActorType           string
	// LotID and LotNumber are the finished-goods lot an approving final release assigned.
	LotID     *string `audit:"lot_id"`
	LotNumber *string `audit:"lot_number"`
	SignedAt  time.Time
}

// ReleaseLot is the finished-goods lot a final release gives a batch.
type ReleaseLot struct {
	ID        string
	LotNumber string
}

// SignerCandidate is an active account user whose role lets them sign in at least one capacity.
type SignerCandidate struct {
	AccountUserID string
	Name          string
	RoleName      string
	DepartmentID  *string
	IsAdmin       bool
	CanSupervise  bool
	CanInspect    bool
}

// EligibleSigner is a person who may sign a step's batches off, and in which capacities.
type EligibleSigner struct {
	AccountUserID string
	Name          string
	RoleName      string
	Capacities    []constants.SignatureCapacity
}

// SignBatchParams is a request to sign a batch at the step it is at.
type SignBatchParams struct {
	BatchID string
	// Type defaults from the step's requirement; only a correction is asked for explicitly.
	Type     *constants.BatchSignatureType
	Decision constants.BatchSignatureDecision
	// Capacity says which of the two signatures a first-article step needs this one is. Elsewhere it
	// follows from the step's requirement.
	Capacity            *constants.SignatureCapacity
	SignerAccountUserID string
	PIN                 string
	Checklist           []ChecklistResponse
	Note                *string
}

// Signer is a person whose signing PIN was just checked, as they were at that moment.
type Signer struct {
	AccountUserID string
	UserID        string
	Name          string
	RoleName      string
	IsAdmin       bool
	DepartmentID  *string
	// Permissions holds "<domain>:<action>" codes; admins hold every permission and have none listed.
	Permissions map[string]bool
}

// HasPermission reports whether the signer holds a permission.
func (s *Signer) HasPermission(domain, action string) bool {
	return s.IsAdmin || s.Permissions[domain+":"+action]
}

// OperatorCheckIn is an operator identifying themselves at a scanning station.
type OperatorCheckIn struct {
	ID                string
	AccountID         string
	ScanningStationID string `audit:"scanning_station_id"`
	AccountUserID     string `audit:"account_user_id"`
	OperatorName      string `audit:"operator_name"`
	CheckedInAt       time.Time
	ExpiresAt         time.Time  `audit:"expires_at"`
	CheckedOutAt      *time.Time `audit:"checked_out_at"`
}

// StatusAt reports whether the check-in still identifies who is working at the station.
func (c *OperatorCheckIn) StatusAt(now time.Time) constants.OperatorCheckInStatus {
	if c.CheckedOutAt != nil || !now.Before(c.ExpiresAt) {
		return constants.OperatorCheckInStatusEnded
	}
	return constants.OperatorCheckInStatusActive
}

// CheckInOperatorParams is an operator checking in at a station with their signing PIN.
type CheckInOperatorParams struct {
	ScanningStationID string
	AccountUserID     string
	PIN               string
}
