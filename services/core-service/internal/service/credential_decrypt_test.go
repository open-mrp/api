package service

import (
	"testing"

	"github.com/open-mrp/api/shared/crypto"
	"github.com/stretchr/testify/require"
)

// testEncryptionKey returns a deterministic 32-byte AES-256 key for round-trip tests.
func testEncryptionKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

// sealShippoCreds encrypts a credentials JSON body exactly the way integration
// credentials are sealed at rest: the enc_v1 AES-256-GCM envelope with the
// account ID as additional authenticated data. This mirrors both this service's
// account_integration_service.go and the legacy dashboard API's
// account-integration.repo.ts, which both pass the account ID as the AAD.
func sealShippoCreds(t *testing.T, key []byte, accountID, credsJSON string) string {
	t.Helper()
	encoded, err := crypto.EncryptAESGCM([]byte(credsJSON), key, []byte(accountID), "k1")
	require.NoError(t, err)
	return encoded
}

// TestDecryptShippoAPIKey_RoundTrip proves the live-rate path can read a Shippo
// credential blob sealed with the account ID as AAD — the on-disk format written
// by both this service and the legacy dashboard API. This is the regression guard
// for the 401 "Token does not exist" bug, where the encrypted blob was handed to
// Shippo verbatim instead of being decrypted first.
func TestDecryptShippoAPIKey_RoundTrip(t *testing.T) {
	key := testEncryptionKey()
	const accountID = "ac_01874vgjg0fbjsyj3ajsyzxhz9"

	encoded := sealShippoCreds(t, key, accountID, `{"api_key":"shippo_live_abc123"}`)

	apiKey, apiErr := decryptShippoAPIKey(encoded, key, accountID)
	require.Nil(t, apiErr)
	require.Equal(t, "shippo_live_abc123", apiKey)
}

// TestDecryptShippoAPIKey_LegacyCamelCaseKey proves credentials persisted by older
// code under the camelCase "apiKey" field are still readable, since ShippoCredentials
// accepts both the canonical snake_case and the legacy key.
func TestDecryptShippoAPIKey_LegacyCamelCaseKey(t *testing.T) {
	key := testEncryptionKey()
	const accountID = "ac_legacy"

	encoded := sealShippoCreds(t, key, accountID, `{"apiKey":"shippo_legacy_xyz"}`)

	apiKey, apiErr := decryptShippoAPIKey(encoded, key, accountID)
	require.Nil(t, apiErr)
	require.Equal(t, "shippo_legacy_xyz", apiKey)
}

// TestDecryptShippoAPIKey_NilAADFails documents the exact defect that was fixed:
// credentials are sealed with the account ID as AAD, so the previous behaviour of
// decrypting with no AAD cannot open them. A blob sealed the correct way must fail
// to decrypt when the AAD is dropped.
func TestDecryptShippoAPIKey_NilAADFails(t *testing.T) {
	key := testEncryptionKey()
	const accountID = "ac_test"

	encoded := sealShippoCreds(t, key, accountID, `{"api_key":"shippo_live_abc123"}`)

	// Decrypting with an empty AAD (the old, broken behaviour) must fail.
	_, err := crypto.DecryptAESGCM(encoded, key, nil)
	require.Error(t, err)
}

// TestDecryptShippoAPIKey_WrongAccountFails proves the AAD binds the ciphertext to
// its account: a blob sealed for one account cannot be decrypted under another,
// preventing cross-account credential reuse.
func TestDecryptShippoAPIKey_WrongAccountFails(t *testing.T) {
	key := testEncryptionKey()

	encoded := sealShippoCreds(t, key, "ac_owner", `{"api_key":"shippo_live_abc123"}`)

	_, apiErr := decryptShippoAPIKey(encoded, key, "ac_attacker")
	require.NotNil(t, apiErr)
}

// TestDecryptShippoAPIKey_Malformed proves a non-envelope blob surfaces a decrypt
// error rather than panicking or returning a bogus key.
func TestDecryptShippoAPIKey_Malformed(t *testing.T) {
	key := testEncryptionKey()

	_, apiErr := decryptShippoAPIKey("not-an-envelope", key, "ac_test")
	require.NotNil(t, apiErr)
}
