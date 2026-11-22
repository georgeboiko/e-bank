package models

const (
	RuleTriggered    = "TRIGGERED"
	RuleNotTriggered = "NOT_TRIGGERED"
	RuleSkipped      = "SKIPPED"
)

type RuleEvaluation struct {
	RuleID int32
	Status string
	Score  int32
	Reason string
}
