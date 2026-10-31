package constants

type FeedbackReportSeverity string

const (
	FeedbackReportSeverityUrgent FeedbackReportSeverity = "urgent"
	FeedbackReportSeverityMedium FeedbackReportSeverity = "medium"
	FeedbackReportSeverityLow    FeedbackReportSeverity = "low"
)

func (s FeedbackReportSeverity) IsValid() bool {
	switch s {
	case FeedbackReportSeverityUrgent, FeedbackReportSeverityMedium, FeedbackReportSeverityLow:
		return true
	default:
		return false
	}
}

func (s FeedbackReportSeverity) EnumValues() []string {
	return []string{string(FeedbackReportSeverityUrgent), string(FeedbackReportSeverityMedium), string(FeedbackReportSeverityLow)}
}
