package analyzer

import (
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

// AnalysisSeverity's zero value is AnalysisSeverityError, so existing call sites that don't set it keep their current behavior.
type AnalysisSeverity int

const (
	AnalysisSeverityError AnalysisSeverity = iota
	AnalysisSeverityWarning
)

type AnalysisError struct {
	Token    token.Token
	Summary  string
	Details  string
	Severity AnalysisSeverity
}

// Error implements error.
func (e AnalysisError) Error() string {
	if e.Token.Source == nil {
		return fmt.Sprintf("%s: %s", e.Summary, e.Details)
	}
	return fmt.Sprintf("%s:%d: %s: %s", e.Token.Source.File, e.Token.Source.Offset, e.Summary, e.Details)
}
