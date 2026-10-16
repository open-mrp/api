package constants

// SalesOrderStatus is where a sales order is in its lifecycle, as the API reports it. Stored as a
// SalesOrderStatusCode; a draft is stored as `estimate`.
type SalesOrderStatus string

const (
	// SalesOrderStatusDraft is an order not yet committed for fulfillment.
	SalesOrderStatusDraft SalesOrderStatus = "draft"
	// SalesOrderStatusIssued is an order being fulfilled.
	SalesOrderStatusIssued SalesOrderStatus = "issued"
	// SalesOrderStatusFulfilled is an order whose every line shipped or was closed.
	SalesOrderStatusFulfilled SalesOrderStatus = "fulfilled"
	// SalesOrderStatusCanceled is an order stopped before anything shipped.
	SalesOrderStatusCanceled SalesOrderStatus = "canceled"
)

func (m SalesOrderStatus) IsValid() bool {
	switch m {
	case SalesOrderStatusDraft, SalesOrderStatusIssued, SalesOrderStatusFulfilled, SalesOrderStatusCanceled:
		return true
	default:
		return false
	}
}

func (m SalesOrderStatus) EnumValues() []string {
	return []string{
		string(SalesOrderStatusDraft),
		string(SalesOrderStatusIssued),
		string(SalesOrderStatusFulfilled),
		string(SalesOrderStatusCanceled),
	}
}

// Code is the stored status code for m.
func (m SalesOrderStatus) Code() SalesOrderStatusCode {
	if m == SalesOrderStatusDraft {
		return SalesOrderStatusCodeEstimate
	}
	return SalesOrderStatusCode(m)
}

// Status is the API status for a stored code.
func (m SalesOrderStatusCode) Status() SalesOrderStatus {
	if m == SalesOrderStatusCodeEstimate {
		return SalesOrderStatusDraft
	}
	return SalesOrderStatus(m)
}

// CommitmentFirmness says whether a customer can accept an order late.
type CommitmentFirmness string

const (
	// CommitmentFirmnessSoft is a date the customer can accept late.
	CommitmentFirmnessSoft CommitmentFirmness = "soft"
	// CommitmentFirmnessHard is a date the customer can't accept late.
	CommitmentFirmnessHard CommitmentFirmness = "hard"
)

func (m CommitmentFirmness) IsValid() bool {
	switch m {
	case CommitmentFirmnessSoft, CommitmentFirmnessHard:
		return true
	default:
		return false
	}
}

func (m CommitmentFirmness) EnumValues() []string {
	return []string{string(CommitmentFirmnessSoft), string(CommitmentFirmnessHard)}
}

// PromisedLeadTimeSource is where the lead time an order was promised on came from.
type PromisedLeadTimeSource string

const (
	// PromisedLeadTimeSourceCustomer is the customer's lead time (its own, its parent's, its group's or the account's).
	PromisedLeadTimeSourceCustomer PromisedLeadTimeSource = "customer"
	// PromisedLeadTimeSourceItem is the longest sales lead time of the order's items, when it exceeds the customer's.
	PromisedLeadTimeSourceItem PromisedLeadTimeSource = "item"
	// PromisedLeadTimeSourceOverride is a lead time set on the order itself.
	PromisedLeadTimeSourceOverride PromisedLeadTimeSource = "override"
)

func (m PromisedLeadTimeSource) IsValid() bool {
	switch m {
	case PromisedLeadTimeSourceCustomer, PromisedLeadTimeSourceItem, PromisedLeadTimeSourceOverride:
		return true
	default:
		return false
	}
}

func (m PromisedLeadTimeSource) EnumValues() []string {
	return []string{
		string(PromisedLeadTimeSourceCustomer),
		string(PromisedLeadTimeSourceItem),
		string(PromisedLeadTimeSourceOverride),
	}
}

func (m *SalesOrderStatus) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}

func (m *CommitmentFirmness) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}

func (m *PromisedLeadTimeSource) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
