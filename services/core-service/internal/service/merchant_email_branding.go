package service

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/ptrutil"
)

// merchantLetterhead is the account's own branding, resolved for a template that has no order behind it to carry it (a purchase-order submission, a statement of account). Templates with an order build the same values through ackData instead.
type merchantLetterhead struct {
	params       map[string]any
	supportEmail string
}

// merchantLetterheadParams resolves an account's letterhead and footer for the shared merchant email partials. Best-effort throughout, matching the order-backed emails: an account or branding lookup that fails yields a blank letterhead rather than blocking the send.
func merchantLetterheadParams(ctx context.Context, repos domain.RepoFactory, branding BrandingAssets, accountID, accountName, emailSubject string) merchantLetterhead {
	out := merchantLetterhead{
		params: map[string]any{
			"account_name":  accountName,
			"year":          time.Now().Format("2006"),
			"email_subject": emailSubject,
		},
	}

	account, _ := repos.NewAccountRepo().GetByID(ctx, accountID)
	if account != nil && account.Branding != nil {
		out.supportEmail = ptrutil.Deref(account.Branding.SupportEmail)
		out.params["logo_url"] = branding.LogoURL(ctx, ptrutil.Deref(account.Branding.LogoURL))
		out.params["account_email"] = out.supportEmail
		out.params["account_website"] = ptrutil.Deref(account.Branding.WebsiteURL)
		out.params["instagram_handle"] = ptrutil.Deref(account.Branding.InstagramHandle)
		out.params["twitter_handle"] = ptrutil.Deref(account.Branding.TwitterHandle)
		out.params["facebook_handle"] = ptrutil.Deref(account.Branding.FacebookHandle)
		out.params["linkedin_handle"] = ptrutil.Deref(account.Branding.LinkedInHandle)
	}
	if account != nil {
		out.params["marketing_blurb"] = accountMarketingBlurbs[account.ID]
	}

	return out
}

// applyMerchantReplyTo points replies at the merchant. Every merchant-facing template invites the reader to reply, but the platform sends from a no-reply address, so without this the reply is discarded and the customer has to chase their supplier by other means. A blank support email leaves SendAs nil rather than setting an address that also cannot receive.
//
// The notification-service overrides this with the account's configured sender reply-to where one exists; this is the fallback for accounts still sending under the platform address.
func applyMerchantReplyTo(data *messaging.EmailSendData, supportEmail string) {
	if data == nil || supportEmail == "" {
		return
	}
	data.SendAs = &supportEmail
}
