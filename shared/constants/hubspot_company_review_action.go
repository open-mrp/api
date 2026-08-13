package constants

// HubspotCompanyReviewAction represents a decision applied to a HubSpot company-match review.
type HubspotCompanyReviewAction string

const (
	// HubspotCompanyReviewActionLink matches the customer to an existing HubSpot company.
	HubspotCompanyReviewActionLink HubspotCompanyReviewAction = "link"
	// HubspotCompanyReviewActionCreateNew creates a new HubSpot company for the customer.
	HubspotCompanyReviewActionCreateNew HubspotCompanyReviewAction = "create_new"
	// HubspotCompanyReviewActionSkip leaves the customer and its orders out of the sync.
	HubspotCompanyReviewActionSkip HubspotCompanyReviewAction = "skip"
)

func (m HubspotCompanyReviewAction) IsValid() bool {
	switch m {
	case HubspotCompanyReviewActionLink, HubspotCompanyReviewActionCreateNew, HubspotCompanyReviewActionSkip:
		return true
	default:
		return false
	}
}

func (m HubspotCompanyReviewAction) EnumValues() []string {
	return []string{
		string(HubspotCompanyReviewActionLink),
		string(HubspotCompanyReviewActionCreateNew),
		string(HubspotCompanyReviewActionSkip),
	}
}
