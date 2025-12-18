package testutil

const (
	// Test IDs
	EntityIDUser        = "usr_test123"
	EntityIDRole        = "rol_test123"
	MissingEntityIDRole = "rol_bad123"
	EntityIDAccount     = "acc_test123"
	// #nosec G101 - These are test constants, not production credentials
	EntityIDAPIKeyValidSandboxMode = "jg6YG319YMfCUaOi5rSSjV"
	// #nosec G101 - These are test constants, not production credentials
	EntityIDAPIKeyValidProdMode = "56ow4Y4vJjVnsmQd9psBhq"
	// #nosec G101 - These are test constants, not production credentials
	EntityIDAPIKeyExpired = "HBucBP2dj4rxA77gW5mEwv"
	// #nosec G101 - These are test constants, not production credentials
	EntityIDAPIKeyInvalid = "3ODXJEG3PqFO1t0AvknIIF"
	// #nosec G101 - These are test constants, not production credentials
	EntityIDAPIKeyBadSecret    = "Idz8Mcd2ErfwHcFzee8Vti"
	EntityIDAPIKeyNeverExpires = "NvrExp123456789012345678"

	// API Keys
	APIKeyValidSandboxMode = "aug_sk_test_" + EntityIDAPIKeyValidSandboxMode + "_eR0LAkxYmlLllMxoTIwMcLls1Nvn1oIk1Z8pSrOhztciaRPPjK"
	APIKeyValidProdMode    = "aug_sk_prod_" + EntityIDAPIKeyValidProdMode + "_BypPtziojf1MpdSwloJ6vGSVhVwrJQOn6WHe84zmMWxmdhdY2f"
	ApiKeyInvalid          = "aug_sk_prod_" + EntityIDAPIKeyInvalid + "_YPYfj0RZxycEabE2hrj9B8AfycwSrUEeGjdLfOF8MfbpcOi0Ff"
	ApiKeyExpired          = "aug_sk_prod_" + EntityIDAPIKeyExpired + "_az1kn9XJsXC7EEU0qNvHT40U9oj8QRwuULxAX20byI2IaroHZi"
	ApiKeyBadSecret        = "aug_sk_prod_" + EntityIDAPIKeyBadSecret + "_DtGA0hknpXDnvNz7SuTF3A4ZrKt4wEfiz5M6mq1xD2Hdt4XVH"
	ApiKeyInvalidChecksum  = "aug_sk_prod_" + EntityIDAPIKeyValidSandboxMode + "_eR0LAkxYmlLllMxoTIwMcs1Nvn1oIk1Z8pSrOhztciTESTIN"
	ApiKeyTooShort         = "aug_sk_prod_" + EntityIDAPIKeyValidSandboxMode + "_eR0LAkxYmlLllMxoTIwMcs1Nvn1oIk1Z8pSrOhztciaRPPjK"
	ApiKeyTooLong          = "aug_sk_prod_" + EntityIDAPIKeyValidSandboxMode + "_eR0LAkxYmlLllMxoTD3IwMcLls1Nvn1oIk1Z8pSrOhztciaRPPjK"
	ApiKeyInvalidPrefix    = "ag_sk_prod_" + EntityIDAPIKeyValidSandboxMode + "_eR0LAkxYmlLllMxoTIwMcLls1Nvn1oIk1Z8pSrOhztciaRPPjK"
	ApiKeyNeverExpires     = "aug_sk_prod_" + EntityIDAPIKeyNeverExpires + "_T7PceMyiMCxYFA18ZMvN7ZHt9VXl5UDWIdgshcEYdFHgbBqjOj"
)
