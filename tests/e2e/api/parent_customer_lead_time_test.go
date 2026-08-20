//go:build e2e

package api_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The rung between a customer and its group: the lead time a head office hands to
// every location beneath it that has not negotiated its own.
//
// The precedence itself is a pure function covered in internal/scheduling; what is
// covered here is that the hierarchy is actually walked when a real customer is
// resolved, and that a lead time set on a parent reaches a child through the API.
//
// The hierarchy is seeded rather than built here because linking a child under a
// parent customer requires acting as the parent account, which an API key never is.

const (
	// leadTimeParentAccountID is the head office committing to 13 days.
	leadTimeParentAccountID = "ac_01e2eltparent00001"
	// leadTimeChildInheritsID inherits it, having set nothing of its own.
	leadTimeChildInheritsID = "ac_01e2eltchild000001"
	// leadTimeChildOverridesID sits under the same parent with its own 5 days.
	leadTimeChildOverridesID = "ac_01e2eltchild000002"
	// leadTimeChildGroupedID sits under the same parent and in a group committing to 21 days.
	leadTimeChildGroupedID = "ac_01e2eltchild000003"
	// leadTimeChildOfSilentParentID sits in that same group, under a parent with no lead time.
	leadTimeChildOfSilentParentID = "ac_01e2eltchild000004"
	// leadTimeMutableParentAccountID is the head office the write-path test edits. Nothing
	// else resolves it or its child, so the edit cannot be seen by a test running alongside.
	leadTimeMutableParentAccountID = "ac_01e2eltparent00003"
	// leadTimeMutableChildAccountID sits under it, in the same 21-day group.
	leadTimeMutableChildAccountID = "ac_01e2eltchild000005"
)

// resolveLeadTime reads the lead time a new order for this customer would be committed to,
// expanding the named sub-objects.
func resolveLeadTime(t *testing.T, customerID string, includes ...string) map[string]any {
	t.Helper()

	path := customersPath + "/" + customerID + customerLeadTimePathSuffix
	if len(includes) > 0 {
		path += "?include=" + strings.Join(includes, ",")
	}
	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "resolving a lead time must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func TestParentCustomerLeadTime_ReachesAChildThatHasNoneOfItsOwn(t *testing.T) {
	t.Parallel()

	resolved := resolveLeadTime(t, leadTimeChildInheritsID, "parent_customer")

	assert.Equal(t, "13", jsonField(resolved, "days"),
		"a child with no lead time of its own is committed to its parent's")
	assert.Equal(t, "parent_customer", jsonField(resolved, "source"))

	parent := jsonObject(resolved, "parent_customer")
	require.NotNil(t, parent, "the parent that decided has to be named: %v", resolved)
	assert.Equal(t, leadTimeParentAccountID, jsonField(parent, "id"))
	assert.Equal(t, "customer", jsonField(parent, "object"),
		"the expanded parent is a whole customer, not a reference")
	assert.NotEmpty(t, jsonField(parent, "name"), "expanding it must be worth doing: %v", parent)
}

// The sub-objects are expandable, so they cost a query only when asked for. A caller that
// wants the number and the rule pays for neither.
func TestParentCustomerLeadTime_ParentIsNullWithoutTheInclude(t *testing.T) {
	t.Parallel()

	resolved := resolveLeadTime(t, leadTimeChildInheritsID)

	assert.Equal(t, "parent_customer", jsonField(resolved, "source"),
		"the rule is reported either way")
	assert.Nil(t, resolved["parent_customer"], "the object itself waits to be asked for")
}

// A location that negotiated its own terms keeps them: inheritance is a fallback, not
// a head office overwriting what its locations agreed.
func TestParentCustomerLeadTime_ChildOverridesIt(t *testing.T) {
	t.Parallel()

	resolved := resolveLeadTime(t, leadTimeChildOverridesID, "parent_customer")

	assert.Equal(t, "5", jsonField(resolved, "days"))
	assert.Equal(t, "customer", jsonField(resolved, "source"))
	assert.Nil(t, resolved["parent_customer"],
		"a child that sets its own lead time inherited nothing, even though it has a parent")
}

// The parent outranks the group. A group is a segment somebody sorted customers into;
// a parent is the account the terms were negotiated with, so a lead time set on a head
// office has to reach its locations whether or not they were also grouped.
func TestParentCustomerLeadTime_OutranksTheAccountGroup(t *testing.T) {
	t.Parallel()

	resolved := resolveLeadTime(t, leadTimeChildGroupedID, "account_group", "parent_customer")

	assert.Equal(t, "13", jsonField(resolved, "days"))
	assert.Equal(t, "parent_customer", jsonField(resolved, "source"))
	assert.Nil(t, resolved["account_group"],
		"the group did not decide, so it must not be named even when asked for")
	require.NotNil(t, jsonObject(resolved, "parent_customer"),
		"the parent that decided has to be named: %v", resolved)
}

// Having a parent is not the same as inheriting from one. A parent that has set no lead
// time must fall through to the group rather than shadow it, or adding a hierarchy would
// silently drop the terms a group already carried.
func TestParentCustomerLeadTime_SilentParentFallsThroughToTheGroup(t *testing.T) {
	t.Parallel()

	resolved := resolveLeadTime(t, leadTimeChildOfSilentParentID, "account_group", "parent_customer")

	assert.Equal(t, "21", jsonField(resolved, "days"))
	assert.Equal(t, "account_group", jsonField(resolved, "source"))
	assert.Nil(t, resolved["parent_customer"],
		"the parent did not decide, so it must not be named even when asked for")

	group := jsonObject(resolved, "account_group")
	require.NotNil(t, group, "the group that decided has to be named: %v", resolved)
	assert.Equal(t, "account_group", jsonField(group, "object"),
		"the expanded group is a whole account group, not a reference")
	assert.NotEmpty(t, jsonField(group, "name"))
}

// An order stamps what the chain resolved at issue, so a parent's lead time has to reach
// the commitment itself and not only the preview.
func TestParentCustomerLeadTime_StampsAnIssuedOrdersCommitment(t *testing.T) {
	t.Parallel()

	order := issueOrderForCustomer(t, leadTimeChildInheritsID, nil)

	assert.Equal(t, "parent_customer", jsonField(order, "lead_time_source"),
		"the order records the rule that produced its ship-by date")

	// The days are asserted against the date rather than against 13: the shipping
	// calendar can pull a ship-by date back onto an open day, and which day that is
	// depends on when the test runs.
	days, err := strconv.Atoi(jsonField(order, "lead_time_days"))
	require.NoError(t, err, "an issued order must carry the days it committed to")
	assert.LessOrEqual(t, days, 13, "a calendar can only pull a ship-by date earlier")
	assert.Equal(t, issuedPlusDays(t, order, days), shipByDate(t, order))
}

// The write path, end to end: a rep sets a lead time on the head office and every location
// under it is committed to it from that moment, and clearing it hands them back. Setting the
// field is the only part of this feature a user actually performs, so it is exercised through
// the API rather than assumed from a seeded row.
func TestParentCustomerLeadTime_SettingItOnTheParentReachesChildren(t *testing.T) {
	t.Parallel()

	// The fixture starts and ends with no lead time on the parent, so the suite can be
	// re-run against the same database without the precondition below already being false.
	t.Cleanup(func() { setCustomerLeadTime(t, leadTimeMutableParentAccountID, nil) })

	before := resolveLeadTime(t, leadTimeMutableChildAccountID)
	require.Equal(t, "account_group", jsonField(before, "source"),
		"precondition: with the parent silent the child is on its group's lead time")

	setCustomerLeadTime(t, leadTimeMutableParentAccountID, ptrInt(9))

	after := resolveLeadTime(t, leadTimeMutableChildAccountID, "parent_customer")
	assert.Equal(t, "9", jsonField(after, "days"),
		"a lead time set on the head office commits its locations")
	assert.Equal(t, "parent_customer", jsonField(after, "source"))

	parent := jsonObject(after, "parent_customer")
	require.NotNil(t, parent, "the parent that decided has to be named: %v", after)
	assert.Equal(t, leadTimeMutableParentAccountID, jsonField(parent, "id"))

	setCustomerLeadTime(t, leadTimeMutableParentAccountID, nil)

	cleared := resolveLeadTime(t, leadTimeMutableChildAccountID)
	assert.Equal(t, "21", jsonField(cleared, "days"))
	assert.Equal(t, "account_group", jsonField(cleared, "source"),
		"with the parent's lead time gone the child falls back to its group")
}

// setCustomerLeadTime sets a customer's own lead time, or clears it with nil.
func setCustomerLeadTime(t *testing.T, customerID string, days *int) {
	t.Helper()

	body := map[string]any{"lead_time_days": nil}
	if days != nil {
		body["lead_time_days"] = *days
	}
	status, respBody, err := apiClient.Patch(customersPath+"/"+customerID, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "setting a lead time must not 5xx: %s", string(respBody))
	requireStatus(t, 200, status, respBody)
}

// A head office resolves its own lead time, not one inherited from itself. The parent is
// joined by relation id, and a join that matched the row to itself would report every
// customer as inheriting.
func TestParentCustomerLeadTime_ParentResolvesItsOwn(t *testing.T) {
	t.Parallel()

	resolved := resolveLeadTime(t, leadTimeParentAccountID, "parent_customer")

	assert.Equal(t, "13", jsonField(resolved, "days"))
	assert.Equal(t, "customer", jsonField(resolved, "source"))
	assert.Nil(t, resolved["parent_customer"], "a head office has no parent to inherit from")
}

// Inherited, not copied: the child's own lead time stays empty, which is what lets the head
// office renegotiate once instead of once per location. A form showing the field as 13 would
// be showing a value that clearing it could not remove.
func TestParentCustomerLeadTime_IsNotCopiedOntoTheChild(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(customersPath+"/"+leadTimeChildInheritsID+"?include=defaults", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	defaults, ok := parseJSON(body)["defaults"].(map[string]any)
	require.True(t, ok, "a customer carries its ordering defaults: %s", string(body))
	assert.Nil(t, defaults["lead_time_days"], "the child sets nothing of its own")

	assert.Equal(t, "13", jsonField(resolveLeadTime(t, leadTimeChildInheritsID), "days"),
		"while still being committed to its parent's lead time")
}

// The order-entry preview runs the same chain the issue path runs, so a rep quoting a child
// account sees the parent's commitment before the order exists.
func TestParentCustomerLeadTime_ShowsInACommitmentQuote(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(quoteCommitmentPath, map[string]any{
		"buyer_account_id": leadTimeChildInheritsID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "quote must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	quote := parseJSON(body)
	assert.Equal(t, "parent_customer", jsonField(quote, "lead_time_source"))
	assert.NotEmpty(t, jsonField(quote, "ship_by_date"), "a quote must name a date: %s", string(body))
}

// A hierarchy is a commercial relationship, so another tenant must not be able to read what a
// competitor's head office committed to by asking about one of its locations.
func TestParentCustomerLeadTime_TenantIsolation(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	status, body, err := clientB.GetListRaw(customersPath+"/"+leadTimeChildInheritsID+customerLeadTimePathSuffix, nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "cross-tenant read must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "tenant B must not resolve tenant A's customer: %s", string(body))
}
