//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Registration flow updates pinned to 1.0.forge-preview.5, where an option list was replaced only when
// its has_<list> flag was true. The transformer in versiontransforms keeps that meaning.

const preview5APIVersion = "1.0.forge-preview.5"

func TestVersionCompat_RegistrationFlows_FlaggedListIsReplaced(t *testing.T) {
	t.Parallel()
	client := apiClient.WithAPIVersion(preview5APIVersion)
	id := createRegistrationFlow(t, client)

	status, flow, body := patchRegistrationFlow(t, client, id, map[string]any{
		"has_payment_term_ids": true,
		"payment_term_ids":     []string{SeedDefaultPaymentTermID},
	})
	requireStatus(t, 200, status, body)
	assert.Equal(t, []string{SeedDefaultPaymentTermID}, optionIDs(t, flow, "payment_term_options"))
}

func TestVersionCompat_RegistrationFlows_UnflaggedListIsIgnored(t *testing.T) {
	t.Parallel()
	client := apiClient.WithAPIVersion(preview5APIVersion)
	id := createRegistrationFlow(t, client)

	status, flow, body := patchRegistrationFlow(t, client, id, map[string]any{
		"payment_term_ids": []string{SeedDefaultPaymentTermID},
	})
	requireStatus(t, 200, status, body)
	assert.Equal(t, []string{SeedPaymentTermID}, optionIDs(t, flow, "payment_term_options"), "preview.5 ignored a list without its flag")

	status, flow, body = patchRegistrationFlow(t, client, id, map[string]any{
		"has_payment_term_ids": false,
		"payment_term_ids":     []string{SeedDefaultPaymentTermID},
	})
	requireStatus(t, 200, status, body)
	assert.Equal(t, []string{SeedPaymentTermID}, optionIDs(t, flow, "payment_term_options"), "preview.5 ignored a list flagged false")
}

func TestVersionCompat_RegistrationFlows_FlagAloneClearsTheList(t *testing.T) {
	t.Parallel()
	client := apiClient.WithAPIVersion(preview5APIVersion)
	id := createRegistrationFlow(t, client)

	status, flow, body := patchRegistrationFlow(t, client, id, map[string]any{"has_payment_term_ids": true})
	requireStatus(t, 200, status, body)
	assert.Empty(t, optionIDs(t, flow, "payment_term_options"), "preview.5 cleared a flagged list that was not sent")
}

// Create never had the flags, so its lists still apply.
func TestVersionCompat_RegistrationFlows_CreateAppliesItsLists(t *testing.T) {
	t.Parallel()
	client := apiClient.WithAPIVersion(preview5APIVersion)
	id := createRegistrationFlow(t, client)

	status, flow, body := patchRegistrationFlow(t, client, id, map[string]any{"name": uniqueName("e2e-flow-renamed")})
	requireStatus(t, 200, status, body)
	assert.Equal(t, []string{SeedPaymentTermID}, optionIDs(t, flow, "payment_term_options"))
}
