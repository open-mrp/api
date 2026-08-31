package stub

import (
	"context"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// EmailIdentityProvider is a no-op EmailIdentityProvider for test mode: it returns placeholder DKIM tokens and reports every domain as already verified so the bridge can be exercised without SES.
type EmailIdentityProvider struct{}

func (p *EmailIdentityProvider) RegisterDomain(_ context.Context, _ string) ([]string, *apierror.APIError) {
	return []string{"stub1._domainkey", "stub2._domainkey", "stub3._domainkey"}, nil
}

func (p *EmailIdentityProvider) DomainVerified(_ context.Context, _ string) (bool, *apierror.APIError) {
	return true, nil
}

func (p *EmailIdentityProvider) DeleteDomain(_ context.Context, _ string) *apierror.APIError {
	return nil
}

func (p *EmailIdentityProvider) SetMailFromDomain(_ context.Context, _, mailFromSubdomain string) (domain.MailFromRecords, *apierror.APIError) {
	return p.MailFromRecordsFor(mailFromSubdomain), nil
}

func (p *EmailIdentityProvider) MailFromRecordsFor(mailFromSubdomain string) domain.MailFromRecords {
	return domain.MailFromRecords{
		Subdomain: mailFromSubdomain,
		MXRecord:  "10 feedback-smtp.us-east-1.amazonses.com",
		SPFRecord: "v=spf1 include:amazonses.com ~all",
	}
}
