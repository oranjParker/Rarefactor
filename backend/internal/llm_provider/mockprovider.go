package llm_provider

import "context"

// MockProvider for free local testing.
type MockProvider struct{}

func (m *MockProvider) Generate(ctx context.Context, prompt string) (string, error) {
	return `{
		"summary": "This is a mock summary for testing.",
		"keywords": ["mock", "test", "data"],
		"questions": ["Is this real?", "Does it work?"]
	}`, nil
}
