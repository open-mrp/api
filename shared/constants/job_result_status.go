package constants

// JobResultStatus is what became of one row of a bulk request: it created a resource,
// updated an existing one, or failed.
type JobResultStatus string

const (
	// JobResultStatusCreated indicates the row produced a newly created resource.
	JobResultStatusCreated JobResultStatus = "created"
	// JobResultStatusUpdated indicates the row updated an existing resource.
	JobResultStatusUpdated JobResultStatus = "updated"
	// JobResultStatusFailed indicates the row was rejected and wrote nothing.
	JobResultStatusFailed JobResultStatus = "failed"
)

func (a JobResultStatus) IsValid() bool {
	switch a {
	case JobResultStatusCreated, JobResultStatusUpdated, JobResultStatusFailed:
		return true
	default:
		return false
	}
}

func (a JobResultStatus) EnumValues() []string {
	return []string{
		string(JobResultStatusCreated),
		string(JobResultStatusUpdated),
		string(JobResultStatusFailed),
	}
}

func (a *JobResultStatus) StringPtr() *string {
	if a == nil {
		return nil
	}
	s := string(*a)
	return &s
}
