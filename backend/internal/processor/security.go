package processor

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/oranjParker/Rarefactor/internal/core"
)

type SecurityProcessor struct {
	Patterns        []*regexp.Regexp
	FailOnViolation bool
}

func NewSecurityProcessor(failOnViolation bool) *SecurityProcessor {
	signatures := []string{
		`(?i)ignore (all )?previous instructions`,
		`(?i)do not (use|mention|follow)`,
		`(?i)---(.*?)END OF PROMPT(.*?)---`,
		`(?i)\[(.*?)INTERNAL(.*?)\]`,
	}

	compiled := make([]*regexp.Regexp, len(signatures))
	for i, s := range signatures {
		compiled[i] = regexp.MustCompile(s)
	}

	return &SecurityProcessor{
		Patterns:        compiled,
		FailOnViolation: failOnViolation,
	}
}

func (p *SecurityProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	content := strings.ToLower(doc.Content)
	hits := 0

	for _, re := range p.Patterns {
		if re.MatchString(content) {
			hits++
		}
	}

	if hits == 0 {
		return []*core.Document[string]{doc}, nil
	}

	newDoc := doc.Clone()
	if newDoc.Metadata == nil {
		newDoc.Metadata = make(map[string]any)
	}
	newDoc.Metadata["security_score"] = hits
	newDoc.Metadata["potential_injection"] = true

	if p.FailOnViolation {
		return nil, fmt.Errorf("%w: found %d suspicious patterns", core.ErrSecurityViolation, hits)
	}

	return []*core.Document[string]{newDoc}, nil
}
