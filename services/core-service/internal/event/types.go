package event

// PurgeAccountDataPayload is the payload for CoreCmdPurgeAccountData messages.
type PurgeAccountDataPayload struct {
	AccountID string `json:"account_id"`
}

// SeedSandboxPayload is the payload for CoreCmdSeedSandbox messages.
type SeedSandboxPayload struct {
	AccountID string `json:"account_id"`
}
