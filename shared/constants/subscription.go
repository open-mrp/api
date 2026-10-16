package constants

// SubscriptionReason is why a person is subscribed to a record.
type SubscriptionReason string

const (
	// SubscriptionReasonCreator: they created the record.
	SubscriptionReasonCreator SubscriptionReason = "creator"
	// SubscriptionReasonEditor: they edited the record.
	SubscriptionReasonEditor SubscriptionReason = "editor"
	// SubscriptionReasonAssignee: they were assigned to the record, such as its sales rep.
	SubscriptionReasonAssignee SubscriptionReason = "assignee"
	// SubscriptionReasonMentioned: they were @mentioned about the record.
	SubscriptionReasonMentioned SubscriptionReason = "mentioned"
	// SubscriptionReasonManual: they followed the record themselves.
	SubscriptionReasonManual SubscriptionReason = "manual"
)

func (r SubscriptionReason) IsValid() bool {
	switch r {
	case SubscriptionReasonCreator, SubscriptionReasonEditor, SubscriptionReasonAssignee, SubscriptionReasonMentioned, SubscriptionReasonManual:
		return true
	}
	return false
}

func (r SubscriptionReason) EnumValues() []string {
	return []string{string(SubscriptionReasonCreator), string(SubscriptionReasonEditor), string(SubscriptionReasonAssignee),
		string(SubscriptionReasonMentioned), string(SubscriptionReasonManual)}
}

// AutoSubscribeTrigger is the preference value that subscribes the reason it names automatically.
func (r SubscriptionReason) AutoSubscribeTrigger() AutoSubscribeTrigger {
	switch r {
	case SubscriptionReasonCreator:
		return AutoSubscribeCreated
	case SubscriptionReasonEditor:
		return AutoSubscribeEdited
	case SubscriptionReasonAssignee:
		return AutoSubscribeAssigned
	case SubscriptionReasonMentioned:
		return AutoSubscribeMentioned
	}
	return ""
}

// RecordSubscriptionStatus is whether a person is notified about a record.
type RecordSubscriptionStatus string

const (
	// RecordSubscriptionStatusSubscribed: notified about the record.
	RecordSubscriptionStatusSubscribed RecordSubscriptionStatus = "subscribed"
	// RecordSubscriptionStatusUnsubscribed: not notified, and not resubscribed automatically.
	RecordSubscriptionStatusUnsubscribed RecordSubscriptionStatus = "unsubscribed"
)

func (s RecordSubscriptionStatus) IsValid() bool {
	return s == RecordSubscriptionStatusSubscribed || s == RecordSubscriptionStatusUnsubscribed
}

func (s RecordSubscriptionStatus) EnumValues() []string {
	return []string{string(RecordSubscriptionStatusSubscribed), string(RecordSubscriptionStatusUnsubscribed)}
}

// AutoSubscribeTrigger is an event that subscribes a person to a record automatically.
type AutoSubscribeTrigger string

const (
	// AutoSubscribeCreated subscribes the person who creates a record.
	AutoSubscribeCreated AutoSubscribeTrigger = "created"
	// AutoSubscribeAssigned subscribes the person assigned to a record.
	AutoSubscribeAssigned AutoSubscribeTrigger = "assigned"
	// AutoSubscribeMentioned subscribes a person @mentioned about a record.
	AutoSubscribeMentioned AutoSubscribeTrigger = "mentioned"
	// AutoSubscribeEdited subscribes a person who edits a record.
	AutoSubscribeEdited AutoSubscribeTrigger = "edited"
)

func (t AutoSubscribeTrigger) IsValid() bool {
	switch t {
	case AutoSubscribeCreated, AutoSubscribeAssigned, AutoSubscribeMentioned, AutoSubscribeEdited:
		return true
	}
	return false
}

func (t AutoSubscribeTrigger) EnumValues() []string {
	return []string{string(AutoSubscribeCreated), string(AutoSubscribeAssigned), string(AutoSubscribeMentioned), string(AutoSubscribeEdited)}
}

// DefaultAutoSubscribe applies when no person, role or account setting chooses.
func DefaultAutoSubscribe() []AutoSubscribeTrigger {
	return []AutoSubscribeTrigger{AutoSubscribeCreated, AutoSubscribeAssigned, AutoSubscribeMentioned}
}

// FollowableRecordTypes are the record types a person can subscribe to. Following a customer or supplier also covers
// its orders and invoices.
var FollowableRecordTypes = []RecordType{
	RecordTypeSalesOrder, RecordTypePurchaseOrder, RecordTypeProductionRun, RecordTypeInvoice, RecordTypeCustomer, RecordTypeSupplier,
}

// Followable reports whether people can subscribe to records of this type.
func (t RecordType) Followable() bool {
	for _, f := range FollowableRecordTypes {
		if f == t {
			return true
		}
	}
	return false
}

func (v *AutoSubscribeTrigger) StringPtr() *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

func (v *RecordSubscriptionStatus) StringPtr() *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

func (v *SubscriptionReason) StringPtr() *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}
