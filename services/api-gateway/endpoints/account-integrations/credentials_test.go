package accountintegrationep

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/field"
)

func TestEncodeCredentials(t *testing.T) {
	t.Parallel()

	stripe := &IntegrationCredentials{PrivateKey: field.Some("sk_test_1"), PublishableKey: field.Some("pk_test_1"), WebhookSecret: field.Some("whsec_1")}
	got, apiErr := stripe.encode(constants.IntegrationCodeStripe)
	require.Nil(t, apiErr)
	assert.JSONEq(t, `{"private_key":"sk_test_1","publishable_key":"pk_test_1","webhook_secret":"whsec_1"}`, got)

	shippo := &IntegrationCredentials{APIKey: field.Some("shippo_test_1")}
	got, apiErr = shippo.encode(constants.IntegrationCodeShippo)
	require.Nil(t, apiErr)
	assert.JSONEq(t, `{"api_key":"shippo_test_1"}`, got)
}

// Every key the provider needs and every key it doesn't take is reported at once.
func TestEncodeCredentials_ReportsEveryMismatch(t *testing.T) {
	t.Parallel()

	creds := &IntegrationCredentials{PrivateKey: field.Some("sk_test_1"), WebhookSecret: field.Some(""), APIKey: field.Some("shippo_test_1")}
	_, apiErr := creds.encode(constants.IntegrationCodeStripe)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	params := map[string]apierror.Code{}
	for _, e := range apiErr.Errors {
		params[e.Param] = e.Code
	}
	assert.Equal(t, map[string]apierror.Code{
		"credentials.publishable_key": apierror.CodeMissingField,
		"credentials.webhook_secret":  apierror.CodeMissingField,
		"credentials.api_key":         apierror.CodeInvalidFormat,
	}, params)
}
