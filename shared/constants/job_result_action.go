package constants

// JobResultAction is what a bulk operation did to the resource one result row names:
// created a new one or updated an existing one.
type JobResultAction string

const (
	// JobResultActionCreated indicates the row produced a newly created resource.
	JobResultActionCreated JobResultAction = "created"
	// JobResultActionUpdated indicates the row updated an existing resource.
	JobResultActionUpdated JobResultAction = "updated"
)

func (a JobResultAction) IsValid() bool {
	switch a {
	case JobResultActionCreated, JobResultActionUpdated:
		return true
	default:
		return false
	}
}

func (a JobResultAction) EnumValues() []string {
	return []string{
		string(JobResultActionCreated),
		string(JobResultActionUpdated),
	}
}

func (a *JobResultAction) StringPtr() *string {
	if a == nil {
		return nil
	}
	s := string(*a)
	return &s
}
