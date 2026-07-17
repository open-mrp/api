package constants

// HubspotSyncRecordAugnoType is the kind of Augno record a sync record maps from.
type HubspotSyncRecordAugnoType string

const (
	// HubspotSyncRecordAugnoTypeCustomer maps an Augno customer to a HubSpot company.
	HubspotSyncRecordAugnoTypeCustomer HubspotSyncRecordAugnoType = "customer"
	// HubspotSyncRecordAugnoTypeContact maps an Augno customer's billing contact to a HubSpot contact.
	HubspotSyncRecordAugnoTypeContact HubspotSyncRecordAugnoType = "contact"
	// HubspotSyncRecordAugnoTypeDeal maps an Augno sales order to a HubSpot deal.
	HubspotSyncRecordAugnoTypeDeal HubspotSyncRecordAugnoType = "deal"
)

func (m HubspotSyncRecordAugnoType) IsValid() bool {
	switch m {
	case HubspotSyncRecordAugnoTypeCustomer, HubspotSyncRecordAugnoTypeContact, HubspotSyncRecordAugnoTypeDeal:
		return true
	default:
		return false
	}
}

func (m HubspotSyncRecordAugnoType) EnumValues() []string {
	return []string{
		string(HubspotSyncRecordAugnoTypeCustomer),
		string(HubspotSyncRecordAugnoTypeContact),
		string(HubspotSyncRecordAugnoTypeDeal),
	}
}

func (m *HubspotSyncRecordAugnoType) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}

// HubspotSyncRecordHubspotType is the HubSpot CRM object a sync record maps to. The values are HubSpot's own object-type names, which is what its API and URLs use.
type HubspotSyncRecordHubspotType string

const (
	HubspotSyncRecordHubspotTypeCompanies HubspotSyncRecordHubspotType = "companies"
	HubspotSyncRecordHubspotTypeContacts  HubspotSyncRecordHubspotType = "contacts"
	HubspotSyncRecordHubspotTypeDeals     HubspotSyncRecordHubspotType = "deals"
)

func (m HubspotSyncRecordHubspotType) IsValid() bool {
	switch m {
	case HubspotSyncRecordHubspotTypeCompanies, HubspotSyncRecordHubspotTypeContacts, HubspotSyncRecordHubspotTypeDeals:
		return true
	default:
		return false
	}
}

func (m HubspotSyncRecordHubspotType) EnumValues() []string {
	return []string{
		string(HubspotSyncRecordHubspotTypeCompanies),
		string(HubspotSyncRecordHubspotTypeContacts),
		string(HubspotSyncRecordHubspotTypeDeals),
	}
}

func (m *HubspotSyncRecordHubspotType) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
