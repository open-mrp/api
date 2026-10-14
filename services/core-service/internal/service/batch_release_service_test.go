package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

func TestSignatureCapacity(t *testing.T) {
	quality := constants.SignatureCapacityQuality
	supervisor := constants.SignatureCapacitySupervisor

	capacity, apiErr := signatureCapacity(constants.ReleaseRequirementFinal, nil)
	require.Nil(t, apiErr)
	assert.Equal(t, quality, capacity)

	capacity, apiErr = signatureCapacity(constants.ReleaseRequirementSupervisor, nil)
	require.Nil(t, apiErr)
	assert.Equal(t, supervisor, capacity)

	_, apiErr = signatureCapacity(constants.ReleaseRequirementSupervisor, &quality)
	require.NotNil(t, apiErr, "a supervisor step is signed in the supervisor capacity only")

	_, apiErr = signatureCapacity(constants.ReleaseRequirementFirstArticle, nil)
	require.NotNil(t, apiErr, "a first-article signature must say which of the two it is")

	capacity, apiErr = signatureCapacity(constants.ReleaseRequirementFirstArticle, &quality)
	require.Nil(t, apiErr)
	assert.Equal(t, quality, capacity)
}

func TestMatchChecklist(t *testing.T) {
	prompts := []string{"Labeling correct?", "Bag attached?"}
	answer := func(prompt string, a constants.ChecklistAnswer) domain.ChecklistResponse {
		return domain.ChecklistResponse{Prompt: prompt, Answer: a}
	}

	matched, apiErr := matchChecklist(prompts, []domain.ChecklistResponse{
		answer("Bag attached?", constants.ChecklistAnswerYes),
		answer("Labeling correct?", constants.ChecklistAnswerNo),
	})
	require.Nil(t, apiErr)
	assert.Equal(t, []domain.ChecklistResponse{
		answer("Labeling correct?", constants.ChecklistAnswerNo),
		answer("Bag attached?", constants.ChecklistAnswerYes),
	}, matched, "answers come back in the policy's order")

	_, apiErr = matchChecklist(prompts, []domain.ChecklistResponse{answer("Labeling correct?", constants.ChecklistAnswerYes)})
	require.NotNil(t, apiErr, "every prompt must be answered")

	_, apiErr = matchChecklist(prompts, []domain.ChecklistResponse{
		answer("Labeling correct?", constants.ChecklistAnswerYes),
		answer("Bag attached?", constants.ChecklistAnswerYes),
		answer("Box taped?", constants.ChecklistAnswerYes),
	})
	require.NotNil(t, apiErr, "an answer to a prompt the step doesn't have means the checklist was stale")

	_, apiErr = matchChecklist(prompts, []domain.ChecklistResponse{
		answer("Labeling correct?", "maybe"),
		answer("Bag attached?", constants.ChecklistAnswerYes),
	})
	require.NotNil(t, apiErr)
}

func TestCheckSignerMay(t *testing.T) {
	knitting, packing := "dp_knit", "dp_pack"
	policy := &domain.ReleasePolicy{DepartmentID: knitting}
	signer := func(department *string, permissions ...string) *domain.Signer {
		held := map[string]bool{}
		for _, p := range permissions {
			held[p] = true
		}
		return &domain.Signer{Name: "Signer", DepartmentID: department, Permissions: held}
	}

	assert.Nil(t, checkSignerMay(signer(&knitting, "batch_releases:create"), constants.SignatureCapacitySupervisor, policy))
	assert.Nil(t, checkSignerMay(signer(nil, "batch_releases:create"), constants.SignatureCapacitySupervisor, policy),
		"a supervisor with no department may sign anywhere")

	apiErr := checkSignerMay(signer(&packing, "batch_releases:create"), constants.SignatureCapacitySupervisor, policy)
	require.NotNil(t, apiErr, "supervisors sign only for their own department")
	assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)

	require.NotNil(t, checkSignerMay(signer(&knitting, "batch_releases:create"), constants.SignatureCapacityQuality, policy))
	assert.Nil(t, checkSignerMay(signer(&packing, "quality_releases:create"), constants.SignatureCapacityQuality, policy),
		"quality signs anywhere")
	assert.Nil(t, checkSignerMay(&domain.Signer{IsAdmin: true, DepartmentID: &packing}, constants.SignatureCapacitySupervisor, policy))
}

func TestNormalizeChecklistPrompts(t *testing.T) {
	prompts, apiErr := normalizeChecklistPrompts([]string{"  Bag attached? "})
	require.Nil(t, apiErr)
	assert.Equal(t, []string{"Bag attached?"}, prompts)

	_, apiErr = normalizeChecklistPrompts([]string{"Bag attached?", "Bag attached?"})
	require.NotNil(t, apiErr)

	_, apiErr = normalizeChecklistPrompts([]string{" "})
	require.NotNil(t, apiErr)
}

func TestReleaseLotNumber(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	assert.Equal(t, "1234-26286", releaseLotNumber(1234, time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)))
	assert.Equal(t, "7-27001", releaseLotNumber(7, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)))
	// Released late on the 13th in New York is already the 14th in UTC; the plant's date wins.
	late := time.Date(2026, 10, 14, 2, 30, 0, 0, time.UTC)
	assert.Equal(t, "1-26286", releaseLotNumber(1, late.In(newYork)))
}

func TestCheckOperatorMaySign(t *testing.T) {
	batch := &domain.Batch{Operator: &domain.LightOperator{AccountUserID: "acus_operator"}}
	operator := &domain.Signer{AccountUserID: "acus_operator", Name: "Operator"}
	someoneElse := &domain.Signer{AccountUserID: "acus_other", Name: "Other"}

	allowed := &domain.ReleasePolicy{OperatorSigning: constants.ReleaseOperatorSigningAllowed}
	forbidden := &domain.ReleasePolicy{OperatorSigning: constants.ReleaseOperatorSigningForbidden}

	assert.Nil(t, checkOperatorMaySign(operator, batch, allowed))
	assert.Nil(t, checkOperatorMaySign(someoneElse, batch, forbidden))
	apiErr := checkOperatorMaySign(operator, batch, forbidden)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
	assert.Nil(t, checkOperatorMaySign(operator, &domain.Batch{}, forbidden), "a batch scanned with nobody checked in has no operator to exclude")
}

func TestCandidateCapacities(t *testing.T) {
	knitting, packing := "dp_knit", "dp_pack"
	policy := &domain.ReleasePolicy{Requirement: constants.ReleaseRequirementFirstArticle, DepartmentID: knitting}

	supervisorElsewhere := domain.SignerCandidate{CanSupervise: true, DepartmentID: &packing}
	supervisorHere := domain.SignerCandidate{CanSupervise: true, DepartmentID: &knitting}
	inspector := domain.SignerCandidate{CanInspect: true, DepartmentID: &packing}
	admin := domain.SignerCandidate{IsAdmin: true}

	assert.False(t, candidateMay(supervisorElsewhere, constants.SignatureCapacitySupervisor, policy))
	assert.True(t, candidateMay(supervisorHere, constants.SignatureCapacitySupervisor, policy))
	assert.False(t, candidateMay(supervisorHere, constants.SignatureCapacityQuality, policy))
	assert.True(t, candidateMay(inspector, constants.SignatureCapacityQuality, policy))
	assert.True(t, candidateMay(admin, constants.SignatureCapacitySupervisor, policy))
	assert.True(t, candidateMay(admin, constants.SignatureCapacityQuality, policy))

	assert.Equal(t, []constants.SignatureCapacity{constants.SignatureCapacitySupervisor, constants.SignatureCapacityQuality}, capacitiesFor(constants.ReleaseRequirementFirstArticle))
	assert.Empty(t, capacitiesFor(constants.ReleaseRequirementNone))
}
