//go:build e2e

package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/id"
)

const releaseChecklistPrompt = "Labeling and packaging correct?"

func releasePolicyPath(departmentID string) string {
	return departmentsPath + "/" + departmentID + "/release-policy"
}

// releaseChain is a scan chain whose stations each sit in a department of their own, so a test can gate
// one department without gating the shared seed department every other scan test uses.
type releaseChain struct {
	scanChain
	initDept, moveDept, splitDept string
}

func newScanStationIn(t *testing.T, stationType, departmentID string) string {
	t.Helper()
	markOutbox(t)
	status, raw, err := apiClient.Post(scanningStationsPath, map[string]any{
		"name":                 uniqueName("e2e-release-" + stationType),
		"type":                 stationType,
		"operator_requirement": "material_check",
		"department_id":        departmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, raw)
	id := jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { apiClient.Delete(scanningStationsPath + "/" + id) })
	return id
}

func newReleaseChain(t *testing.T) releaseChain {
	t.Helper()
	department := func() string {
		return jsonField(createAndCleanup(t, departmentsPath, map[string]any{"name": uniqueName("e2e-release-dept")}), "id")
	}
	c := releaseChain{initDept: department(), moveDept: department(), splitDept: department()}
	c.material = newScanMaterial(t)
	c.a = newScanPart(t, scanShippingCategory)
	c.b = newScanPart(t, scanShippingCategory)
	c.c = newScanPart(t, scanShippingCategory)
	c.initStation = newScanStationIn(t, "init_batch", c.initDept)
	c.moveStation = newScanStationIn(t, "move_batch", c.moveDept)
	c.splitStation = newScanStationIn(t, "split_batch", c.splitDept)
	c.initStep = newScanStep(t, c.initStation, scanQty{c.a.id, "1", unitEach},
		scanConsumption{itemID: c.material.id, value: "2", unitID: unitEach})
	c.moveStep = newScanStep(t, c.moveStation, scanQty{c.b.id, "1", unitEach},
		scanConsumption{itemID: c.a.id, value: "1", unitID: unitEach},
		scanConsumption{itemID: c.material.id, value: "1", unitID: unitEach})
	c.splitStep = newScanStep(t, c.splitStation, scanQty{c.c.id, "1", unitEach},
		scanConsumption{itemID: c.b.id, value: "1", unitID: unitEach})
	stockScanItem(t, c.material.id, "1000", unitEach)
	return c
}

func batchSignaturesPath(batchID string) string {
	return batchesPath + "/" + batchID + "/signatures"
}

func operatorCheckInsPath(stationID string) string {
	return scanningStationsPath + "/" + stationID + "/operator-check-ins"
}

// signer is a person who can sign batches off: an account user under a role holding the given
// permission codes, with a signing PIN they have set.
type signer struct {
	accountUserID string
	pin           string
}

// newSigner creates an operator-style account user (no email) under a role with the given
// permissions, gives them a temporary PIN and changes it, as a person would at a station.
func newSigner(t *testing.T, permissionCodes ...string) signer {
	t.Helper()
	db := authDB(t)
	suffix := uuid.New().String()[:12]
	roleID := "rl_e2esigner_" + suffix
	_, err := db.Exec(`INSERT INTO role (id, name, role_type_code, account_id, created_at, updated_at) VALUES (?, ?, 'user', ?, NOW(3), NOW(3))`,
		roleID, "E2E signer "+suffix, SeedAccountID)
	require.NoError(t, err)
	for i, code := range append([]string{"batches"}, permissionCodes...) {
		_, err = db.Exec("INSERT INTO role_permission (id, `create`, `read`, `update`, `delete`, role_id, permission_code, created_at, updated_at) VALUES (?, 1, 1, 0, 0, ?, ?, NOW(3), NOW(3))",
			fmt.Sprintf("rlpm_e2esigner_%s_%d", suffix, i), roleID, code)
		require.NoError(t, err)
	}

	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"username": uniqueName("e2e-signer"),
		"password": "OperatorPass123!",
		"role_id":  roleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, body)
	accountUserID := jsonField(parseJSON(body), "id")
	// A user without an email always gets the scanner role; roles are reassigned in SQL, as an operator would.
	_, err = db.Exec("UPDATE account_user SET role_id = ? WHERE id = ?", roleID, accountUserID)
	require.NoError(t, err)
	t.Cleanup(func() {
		removeAccountUser(accountUserID)
		_, _ = db.Exec("DELETE FROM role_permission WHERE role_id = ?", roleID)
		_, _ = db.Exec("DELETE FROM role WHERE id = ?", roleID)
	})

	status, body, err = apiClient.Post(setTemporarySigningPinPath(accountUserID), map[string]any{"pin": "4826"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, body)
	status, body = changeSigningPin(t, apiClient, accountUserID, "4826", "7391")
	requireStatus(t, http.StatusOK, status, body)
	return signer{accountUserID: accountUserID, pin: "7391"}
}

func putReleasePolicy(t *testing.T, departmentID string, body map[string]any) map[string]any {
	t.Helper()
	status, raw, err := apiClient.Put(releasePolicyPath(departmentID), body)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	return parseJSON(raw)
}

// gateDepartment sets a department's release policy for the rest of the test. The returned func puts it back to
// none; defer it so batches can be undone by the scans' cleanups, which run after deferred calls.
func gateDepartment(t *testing.T, departmentID, requirement, enforcement string, checklist ...string) func() {
	t.Helper()
	body := map[string]any{"requirement": requirement, "enforcement": enforcement}
	if len(checklist) > 0 {
		body["checklist"] = checklist
	}
	putReleasePolicy(t, departmentID, body)
	return func() { putReleasePolicy(t, departmentID, map[string]any{"requirement": "none"}) }
}

func sign(t *testing.T, batchID string, s signer, body map[string]any) (int, map[string]any, []byte) {
	t.Helper()
	body["signer_account_user_id"] = s.accountUserID
	if _, ok := body["pin"]; !ok {
		body["pin"] = s.pin
	}
	status, raw, err := apiClient.Post(batchSignaturesPath(batchID), body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "signing must not 5xx: %s", string(raw))
	return status, parseJSON(raw), raw
}

func mustSign(t *testing.T, batchID string, s signer, body map[string]any) map[string]any {
	t.Helper()
	status, got, raw := sign(t, batchID, s, body)
	requireStatus(t, http.StatusCreated, status, raw)
	return got
}

func checklistAnswers(answer string, prompts ...string) []map[string]any {
	answers := make([]map[string]any, len(prompts))
	for i, prompt := range prompts {
		answers[i] = map[string]any{"prompt": prompt, "answer": answer}
	}
	return answers
}

// --- Release policy ---

func TestBatchReleases_PolicyDefaultsAndReplace(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)

	status, raw, err := apiClient.GetListRaw(releasePolicyPath(c.initDept), nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	got := parseJSON(raw)
	assertObjectField(t, got, "release_policy")
	assert.Equal(t, c.initDept, jsonField(jsonObject(got, "department"), "id"))
	assert.NotEmpty(t, jsonField(jsonObject(got, "department"), "name"))
	assert.Equal(t, "none", jsonField(got, "requirement"))
	assert.Equal(t, "block", jsonField(got, "enforcement"))
	assertNilField(t, got, "rationale")
	assert.Empty(t, jsonArray(jsonObject(got, "checklist"), "data"))
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	got = putReleasePolicy(t, c.initDept, map[string]any{
		"requirement": "supervisor",
		"enforcement": "warn",
		"rationale":   "  Set-up errors make bad product.  ",
		"checklist":   []string{" " + releaseChecklistPrompt, "Bag attached?"},
	})
	assert.Equal(t, "supervisor", jsonField(got, "requirement"))
	assert.Equal(t, "warn", jsonField(got, "enforcement"))
	assert.Equal(t, "Set-up errors make bad product.", jsonField(got, "rationale"))
	prompts := jsonArray(jsonObject(got, "checklist"), "data")
	require.Len(t, prompts, 2)
	assert.Equal(t, releaseChecklistPrompt, prompts[0])

	// A replace with only the requirement resets everything else to its default.
	got = putReleasePolicy(t, c.initDept, map[string]any{"requirement": "none"})
	assert.Equal(t, "block", jsonField(got, "enforcement"))
	assertNilField(t, got, "rationale")
	assert.Empty(t, jsonArray(jsonObject(got, "checklist"), "data"))
}

func TestBatchReleases_PolicyValidation(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)

	for name, body := range map[string]map[string]any{
		"unknown requirement": {"requirement": "manager"},
		"missing requirement": {"enforcement": "block"},
		"duplicate prompts":   {"requirement": "quality", "checklist": []string{"Bag attached?", "Bag attached?"}},
		"blank prompt":        {"requirement": "quality", "checklist": []string{"  "}},
	} {
		status, raw, err := apiClient.Put(releasePolicyPath(c.initDept), body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, status, "%s: %s", name, string(raw))
	}

	status, raw, err := apiClient.Put(releasePolicyPath("dp_doesnotexist"), map[string]any{"requirement": "quality"})
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, raw)
}

// --- Signing and scanning ---

func TestBatchReleases_BlockingStepHoldsTheScanUntilSigned(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)
	supervisor := newSigner(t, "batch_releases")
	notASupervisor := newSigner(t)
	defer gateDepartment(t, c.initDept, "supervisor", "block", releaseChecklistPrompt)()

	batch := planScanBatch(t, c.a.id, "5", unitEach)
	initializeScan(t, batch, c.initStation, nil)

	status, _, raw := postScan(t, "move", map[string]any{"batch_ids": []string{batch}, "production_step_id": c.moveStep, "scanning_station_id": c.moveStation})
	requireAPIError(t, http.StatusConflict, "batch_release_required", status, raw)

	status, _, raw = sign(t, batch, notASupervisor, map[string]any{"decision": "approved", "checklist": checklistAnswers("yes", releaseChecklistPrompt)})
	requireAPIError(t, http.StatusForbidden, "insufficient_permissions", status, raw)

	status, _, raw = sign(t, batch, supervisor, map[string]any{"decision": "approved", "pin": "0000", "checklist": checklistAnswers("yes", releaseChecklistPrompt)})
	requireAPIError(t, http.StatusForbidden, "signing_pin_invalid", status, raw)

	status, _, raw = sign(t, batch, supervisor, map[string]any{"decision": "approved"})
	requireStatus(t, http.StatusBadRequest, status, raw)

	signed := mustSign(t, batch, supervisor, map[string]any{"decision": "approved", "checklist": checklistAnswers("yes", releaseChecklistPrompt)})
	assertIDFormat(t, jsonField(signed, "id"), id.BatchSignatureIDPrefix)
	assertObjectField(t, signed, "batch_signature")
	assert.Equal(t, batch, jsonField(jsonObject(signed, "batch"), "id"))
	assert.Equal(t, c.initStep, jsonField(jsonObject(signed, "production_step"), "id"))
	assert.NotNil(t, jsonObject(signed, "production_run"))
	assert.Equal(t, "step_release", jsonField(signed, "type"))
	assert.Equal(t, "approved", jsonField(signed, "decision"))
	assert.Equal(t, "supervisor", jsonField(signed, "capacity"))
	assert.Equal(t, supervisor.accountUserID, jsonField(jsonObject(signed, "signer"), "id"))
	assert.NotEmpty(t, jsonField(jsonObject(signed, "signer"), "name"))
	assert.NotEmpty(t, jsonField(signed, "signer_role"))
	answers := jsonArray(jsonObject(signed, "checklist"), "data")
	require.Len(t, answers, 1)
	assert.Equal(t, releaseChecklistPrompt, jsonField(answers[0].(map[string]any), "prompt"))
	assert.Equal(t, "yes", jsonField(answers[0].(map[string]any), "answer"))
	assertNilField(t, signed, "note")
	assert.NotNil(t, jsonObject(signed, "actor"))
	assertValidTimestamp(t, jsonField(signed, "signed_at"), "signed_at")
	assert.NotContains(t, string(mustJSON(t, signed)), supervisor.pin, "the PIN never comes back")

	moveScan(t, c.moveStation, c.moveStep, batch)

	status, raw, err := apiClient.GetListRaw(batchSignaturesPath(batch), nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	listed := jsonArray(parseJSON(raw), "data")
	require.Len(t, listed, 1)
	assert.Equal(t, jsonField(signed, "id"), jsonField(listed[0].(map[string]any), "id"))
}

func TestBatchReleases_RejectionHoldsEvenAWarnStep(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)
	inspector := newSigner(t, "quality_releases")
	defer gateDepartment(t, c.initDept, "quality", "warn")()

	batch := planScanBatch(t, c.a.id, "5", unitEach)
	initializeScan(t, batch, c.initStation, nil)

	status, _, raw := sign(t, batch, inspector, map[string]any{"decision": "rejected"})
	requireStatus(t, http.StatusBadRequest, status, raw)
	rejected := mustSign(t, batch, inspector, map[string]any{"decision": "rejected", "note": "Wrong yarn on feeder 3."})
	assert.Equal(t, "quality", jsonField(rejected, "capacity"))
	assert.Equal(t, "Wrong yarn on feeder 3.", jsonField(rejected, "note"))

	status, _, raw = postScan(t, "move", map[string]any{"batch_ids": []string{batch}, "production_step_id": c.moveStep, "scanning_station_id": c.moveStation})
	requireAPIError(t, http.StatusConflict, "batch_on_hold", status, raw)

	mustSign(t, batch, inspector, map[string]any{"decision": "approved", "note": "Re-inspected after re-threading."})
	moveScan(t, c.moveStation, c.moveStep, batch)
}

func TestBatchReleases_FinalReleaseNeedsEveryEarlierSignOff(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)
	supervisor := newSigner(t, "batch_releases")
	inspector := newSigner(t, "quality_releases")
	defer gateDepartment(t, c.initDept, "supervisor", "warn")()
	defer gateDepartment(t, c.moveDept, "final", "block", releaseChecklistPrompt)()

	batch := planScanBatch(t, c.a.id, "5", unitEach)
	initializeScan(t, batch, c.initStation, nil)
	// The warn step lets the unsigned batch through.
	moved := moveScan(t, c.moveStation, c.moveStep, batch)
	movedID := jsonField(moved, "id")

	status, _, raw := sign(t, movedID, inspector, map[string]any{"decision": "approved", "checklist": checklistAnswers("yes", releaseChecklistPrompt)})
	requireAPIError(t, http.StatusConflict, "batch_release_required", status, raw)

	// The supervisor signs the earlier batch after the fact; it stays on its record.
	mustSign(t, batch, supervisor, map[string]any{"decision": "approved"})

	final := mustSign(t, movedID, inspector, map[string]any{"decision": "approved", "checklist": checklistAnswers("yes", releaseChecklistPrompt)})
	assert.Equal(t, "final_release", jsonField(final, "type"))
	assert.Equal(t, "quality", jsonField(final, "capacity"))
	lot := jsonObject(final, "lot")
	require.NotNil(t, lot, "an approving final release assigns the lot")
	lotNumber := jsonField(lot, "name")
	assert.Regexp(t, `^\d+-\d{5}$`, lotNumber, "sequence and Julian date")
	assert.True(t, strings.HasSuffix(lotNumber, fmt.Sprintf("-%02d%03d", time.Now().UTC().Year()%100, time.Now().UTC().YearDay())) ||
		strings.HasSuffix(lotNumber, fmt.Sprintf("-%02d%03d", time.Now().UTC().Add(-time.Hour).Year()%100, time.Now().UTC().Add(-time.Hour).YearDay())))

	// The batch carries the lot among its lots.
	flow := flowByBatch(t, readAs(t, apiClient, batchesPath+"/"+movedID+"/flow", nil))
	var releaseLots []string
	for _, l := range jsonArray(jsonObject(jsonObject(flow[movedID].(map[string]any), "batch"), "lots"), "data") {
		if jsonField(l.(map[string]any), "type") == "release" {
			releaseLots = append(releaseLots, jsonField(l.(map[string]any), "lot_number"))
		}
	}
	assert.Equal(t, []string{lotNumber}, releaseLots)

	// Withdrawn and released again, the batch keeps its first lot.
	mustSign(t, movedID, inspector, map[string]any{"decision": "rejected", "note": "Recount the bag.", "checklist": checklistAnswers("no", releaseChecklistPrompt)})
	again := mustSign(t, movedID, inspector, map[string]any{"decision": "approved", "checklist": checklistAnswers("yes", releaseChecklistPrompt)})
	assert.Equal(t, lotNumber, jsonField(jsonObject(again, "lot"), "name"))

	// A supervisor can't give the final release.
	status, _, raw = sign(t, movedID, supervisor, map[string]any{"decision": "approved", "checklist": checklistAnswers("yes", releaseChecklistPrompt)})
	requireAPIError(t, http.StatusForbidden, "insufficient_permissions", status, raw)
}

func TestBatchReleases_FirstArticleCoversTheRun(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)
	supervisor := newSigner(t, "batch_releases")
	inspector := newSigner(t, "quality_releases")
	defer gateDepartment(t, c.initDept, "first_article", "block")()

	run := createRunWithBatches(t,
		map[string]any{"item_id": c.a.id, "quantity_value": "2", "quantity_unit_id": unitEach},
		map[string]any{"item_id": c.a.id, "quantity_value": "2", "quantity_unit_id": unitEach})
	batches := runBatches(t, jsonField(run, "id"), "run")
	require.Len(t, batches, 2)
	first, second := jsonField(batches[0], "id"), jsonField(batches[1], "id")
	initializeScan(t, first, c.initStation, nil)
	initializeScan(t, second, c.initStation, nil)

	status, _, raw := sign(t, first, supervisor, map[string]any{"decision": "approved"})
	requireAPIError(t, http.StatusBadRequest, "validation_failed", status, raw)
	mustSign(t, first, supervisor, map[string]any{"decision": "approved", "capacity": "supervisor"})

	status, _, raw = postScan(t, "move", map[string]any{"batch_ids": []string{second}, "production_step_id": c.moveStep, "scanning_station_id": c.moveStation})
	requireAPIError(t, http.StatusConflict, "batch_release_required", status, raw)

	mustSign(t, first, inspector, map[string]any{"decision": "approved", "capacity": "quality"})
	// The set-up verified on the first batch covers the run's second.
	moveScan(t, c.moveStation, c.moveStep, second)
	moveScan(t, c.moveStation, c.moveStep, first)
}

// --- Corrections ---

func TestBatchReleases_UndoAtAGatedStepNeedsACorrection(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)
	supervisor := newSigner(t, "batch_releases")
	defer gateDepartment(t, c.initDept, "supervisor", "block")()

	batch := planScanBatch(t, c.a.id, "5", unitEach)
	initializeScan(t, batch, c.initStation, nil)
	settleScans(t, batch)

	status, raw, err := apiClient.Delete(batchesPath + "/" + batch)
	require.NoError(t, err)
	requireAPIError(t, http.StatusConflict, "correction_signature_required", status, raw)

	status, _, raw = sign(t, batch, supervisor, map[string]any{"type": "correction", "decision": "approved"})
	requireStatus(t, http.StatusBadRequest, status, raw)
	correction := mustSign(t, batch, supervisor, map[string]any{"type": "correction", "decision": "approved", "note": "Scanned at the wrong station."})
	assert.Equal(t, "correction", jsonField(correction, "type"))
	assert.Empty(t, jsonArray(jsonObject(correction, "checklist"), "data"))

	status, raw, err = apiClient.Delete(batchesPath + "/" + batch)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)

	// The signature outlives the scan it corrected.
	status, raw, err = apiClient.GetListRaw(batchSignaturesPath(batch), nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	assert.Len(t, jsonArray(parseJSON(raw), "data"), 1)
}

func TestBatchReleases_ReleasedBatchCantBeUndone(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)
	supervisor := newSigner(t, "batch_releases")
	defer gateDepartment(t, c.initDept, "supervisor", "block")()

	batch := planScanBatch(t, c.a.id, "5", unitEach)
	initializeScan(t, batch, c.initStation, nil)
	settleScans(t, batch)
	mustSign(t, batch, supervisor, map[string]any{"decision": "approved"})
	mustSign(t, batch, supervisor, map[string]any{"type": "correction", "decision": "approved", "note": "Wrong quantity."})

	status, raw, err := apiClient.Delete(batchesPath + "/" + batch)
	require.NoError(t, err)
	requireAPIError(t, http.StatusConflict, "resource_conflict", status, raw)

	// Withdrawing the release lets the correction stand.
	mustSign(t, batch, supervisor, map[string]any{"decision": "rejected", "note": "Withdrawn to correct the quantity."})
	status, raw, err = apiClient.Delete(batchesPath + "/" + batch)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
}

// --- Operator check-in ---

func TestBatchReleases_OperatorCheckIn(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)
	operator := newSigner(t)

	status, raw, err := apiClient.GetListRaw(scanningStationsPath+"/"+c.initStation, nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	assert.Equal(t, "optional", jsonField(parseJSON(raw), "operator_check_in"))

	status, raw, err = apiClient.Patch(scanningStationsPath+"/"+c.initStation, map[string]any{"operator_check_in": "required"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	assert.Equal(t, "required", jsonField(parseJSON(raw), "operator_check_in"))

	batch := planScanBatch(t, c.a.id, "5", unitEach)
	status, _, raw = postScan(t, "initialize", map[string]any{"batch_id": batch, "scanning_station_id": c.initStation})
	requireAPIError(t, http.StatusConflict, "operator_check_in_required", status, raw)

	status, raw, err = apiClient.GetListRaw(scanningStationsPath+"/"+c.initStation+"/operator-check-in", nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, raw)

	status, raw, err = apiClient.Post(operatorCheckInsPath(c.initStation), map[string]any{"account_user_id": operator.accountUserID, "pin": "1111"}, newIdempotencyKey())
	require.NoError(t, err)
	requireAPIError(t, http.StatusForbidden, "signing_pin_invalid", status, raw)

	status, raw, err = apiClient.Post(operatorCheckInsPath(c.initStation), map[string]any{"account_user_id": operator.accountUserID, "pin": operator.pin}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, raw)
	checkIn := parseJSON(raw)
	assertIDFormat(t, jsonField(checkIn, "id"), id.OperatorCheckInIDPrefix)
	assertObjectField(t, checkIn, "operator_check_in")
	assert.Equal(t, "active", jsonField(checkIn, "status"))
	assert.Equal(t, operator.accountUserID, jsonField(jsonObject(checkIn, "operator"), "id"))
	assert.Equal(t, c.initStation, jsonField(jsonObject(checkIn, "scanning_station"), "id"))
	assertValidTimestamp(t, jsonField(checkIn, "checked_in_at"), "checked_in_at")
	assertValidTimestamp(t, jsonField(checkIn, "expires_at"), "expires_at")
	assertNilField(t, checkIn, "checked_out_at")

	status, raw, err = apiClient.GetListRaw(scanningStationsPath+"/"+c.initStation+"/operator-check-in", nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	assert.Equal(t, jsonField(checkIn, "id"), jsonField(parseJSON(raw), "id"))

	initializeScan(t, batch, c.initStation, nil)
	flow := flowByBatch(t, readAs(t, apiClient, batchesPath+"/"+batch+"/flow", nil))
	scanned := jsonObject(flow[batch].(map[string]any), "batch")
	require.NotNil(t, jsonObject(scanned, "operator"), "the scan records who was checked in")
	assert.Equal(t, operator.accountUserID, jsonField(jsonObject(scanned, "operator"), "id"))

	status, raw, err = apiClient.Post(operatorCheckInsPath(c.initStation)+"/"+jsonField(checkIn, "id")+"/actions/check-out", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	ended := parseJSON(raw)
	assert.Equal(t, "ended", jsonField(ended, "status"))
	assertValidTimestamp(t, jsonField(ended, "checked_out_at"), "checked_out_at")

	status, raw, err = apiClient.GetListRaw(scanningStationsPath+"/"+c.initStation+"/operator-check-in", nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusNotFound, status, raw)
}

func TestBatchReleases_CheckInReplacesThePreviousOperator(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)
	first, second := newSigner(t), newSigner(t)

	for _, op := range []signer{first, second} {
		status, raw, err := apiClient.Post(operatorCheckInsPath(c.initStation), map[string]any{"account_user_id": op.accountUserID, "pin": op.pin}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, http.StatusCreated, status, raw)
	}

	status, raw, err := apiClient.GetListRaw(scanningStationsPath+"/"+c.initStation+"/operator-check-in", nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	assert.Equal(t, second.accountUserID, jsonField(jsonObject(parseJSON(raw), "operator"), "id"))
	assert.False(t, strings.Contains(string(raw), first.accountUserID))
}

// --- Separation of duties and eligible signers ---

func TestBatchReleases_OperatorCantSignTheirOwnBatchWhenForbidden(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)
	operatorSupervisor := newSigner(t, "batch_releases")
	otherSupervisor := newSigner(t, "batch_releases")
	inspector := newSigner(t, "quality_releases")
	putReleasePolicy(t, c.initDept, map[string]any{"requirement": "supervisor", "operator_signing": "forbidden"})
	defer putReleasePolicy(t, c.initDept, map[string]any{"requirement": "none"})

	status, raw, err := apiClient.Post(operatorCheckInsPath(c.initStation), map[string]any{"account_user_id": operatorSupervisor.accountUserID, "pin": operatorSupervisor.pin}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, http.StatusCreated, status, raw)

	batch := planScanBatch(t, c.a.id, "5", unitEach)
	initializeScan(t, batch, c.initStation, nil)

	status, _, raw = sign(t, batch, operatorSupervisor, map[string]any{"decision": "approved"})
	requireAPIError(t, http.StatusForbidden, "insufficient_permissions", status, raw)

	// The signer picker leaves the operator out for this batch, and the inspector out of a supervisor step.
	status, raw, err = apiClient.GetListRaw(batchesPath+"/"+batch+"/eligible-signers", nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	ids := map[string][]any{}
	for _, row := range jsonArray(parseJSON(raw), "data") {
		signerRow := row.(map[string]any)
		assert.Equal(t, "eligible_signer", jsonField(signerRow, "object"))
		ids[jsonField(jsonObject(signerRow, "account_user"), "id")] = signerRow["capacities"].([]any)
	}
	assert.NotContains(t, ids, operatorSupervisor.accountUserID)
	assert.NotContains(t, ids, inspector.accountUserID)
	require.Contains(t, ids, otherSupervisor.accountUserID)
	assert.Equal(t, []any{"supervisor"}, ids[otherSupervisor.accountUserID])

	// Where the operator may sign, they're listed.
	putReleasePolicy(t, c.initDept, map[string]any{"requirement": "supervisor", "operator_signing": "allowed"})
	status, raw, err = apiClient.GetListRaw(batchesPath+"/"+batch+"/eligible-signers", nil)
	require.NoError(t, err)
	requireStatus(t, http.StatusOK, status, raw)
	assert.Contains(t, string(raw), operatorSupervisor.accountUserID)
	putReleasePolicy(t, c.initDept, map[string]any{"requirement": "supervisor", "operator_signing": "forbidden"})

	mustSign(t, batch, otherSupervisor, map[string]any{"decision": "approved"})
	moveScan(t, c.moveStation, c.moveStep, batch)
}

func TestBatchReleases_PolicyOperatorSigningDefaultsToAllowed(t *testing.T) {
	t.Parallel()
	c := newReleaseChain(t)

	got := putReleasePolicy(t, c.initDept, map[string]any{"requirement": "quality"})
	assert.Equal(t, "allowed", jsonField(got, "operator_signing"))
	got = putReleasePolicy(t, c.initDept, map[string]any{"requirement": "quality", "operator_signing": "forbidden"})
	assert.Equal(t, "forbidden", jsonField(got, "operator_signing"))
	putReleasePolicy(t, c.initDept, map[string]any{"requirement": "none"})
}
