package aws

import (
	"context"

	"github.com/augno/api/services/notification-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
)

var sesIdentityTracer = tracing.GetTracer("notification-service.aws.ses_identity")

// NewSESIdentityProvider returns the SES-backed implementation of the self-serve domain verification flow. It reuses the v1 SES API already vendored for outbound sending.
func NewSESIdentityProvider(ctx context.Context, region string) (domain.EmailIdentityProvider, *apierror.APIError) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to load AWS configuration.")
	}
	return &sesIdentityProviderImpl{client: ses.NewFromConfig(cfg)}, nil
}

type sesIdentityProviderImpl struct {
	client *ses.Client
}

func (s *sesIdentityProviderImpl) RegisterDomain(ctx context.Context, domainName string) ([]string, *apierror.APIError) {
	ctx, span := sesIdentityTracer.Start(ctx, "aws.ses_identity.register_domain")
	defer span.End()

	// Registering the identity is idempotent; re-verifying an existing domain just re-issues the token.
	if _, err := s.client.VerifyDomainIdentity(ctx, &ses.VerifyDomainIdentityInput{Domain: &domainName}); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to register the email domain with SES."))
	}
	out, err := s.client.VerifyDomainDkim(ctx, &ses.VerifyDomainDkimInput{Domain: &domainName})
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to enable DKIM for the email domain."))
	}
	return out.DkimTokens, nil
}

func (s *sesIdentityProviderImpl) DomainVerified(ctx context.Context, domainName string) (bool, *apierror.APIError) {
	ctx, span := sesIdentityTracer.Start(ctx, "aws.ses_identity.domain_verified")
	defer span.End()

	out, err := s.client.GetIdentityDkimAttributes(ctx, &ses.GetIdentityDkimAttributesInput{Identities: []string{domainName}})
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check the email domain verification status."))
	}
	attr, ok := out.DkimAttributes[domainName]
	if !ok {
		return false, nil
	}
	return attr.DkimVerificationStatus == sestypes.VerificationStatusSuccess, nil
}
