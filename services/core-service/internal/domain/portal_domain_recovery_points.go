package domain

// RecoveryPoint constants for portal domain operations. The provider registration is a foreign mutation (Vercel API), so it gets its own atomic phase: a request that crashes after registering with the provider but before persisting the DNS records resumes here and safely repeats the idempotent provider calls.
const (
	PortalDomainRecoveryPointProviderRegistered RecoveryPoint = "core:portal_domain_provider_registered"
)
