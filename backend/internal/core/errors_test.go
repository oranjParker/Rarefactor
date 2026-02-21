package core

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		err          error
		expected     bool
		expectedWait bool
		name         string
	}{
		{ErrDelayRequired, true, true, "Politeness delay should be retryable with wait duration"},
		{ErrQuotaExceeded, false, false, "Quota exceeded is usually a permanent stop for that job"},
		{ErrRobotsDisallowed, false, false, "Robots disallowed is a permanent policy block"},
		{ErrSecurityViolation, false, false, "Security violation should never be retried"},
		{errors.New("random error"), true, false, "Unknown errors should be retried by default (safe bet)"},
		{fmt.Errorf("wrapped: %w", ErrDelayRequired), true, true, "Wrapped retryable errors should be detected with wait"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRetry, gotWait := IsRetryable(tt.err)

			if gotRetry != tt.expected {
				t.Errorf("IsRetryable() retry flag for %s = %v, want %v", tt.name, gotRetry, tt.expected)
			}

			if tt.expectedWait && gotWait <= 0 {
				t.Errorf("IsRetryable() wait duration for %s expected > 0, got %v", tt.name, gotWait)
			}
		})
	}
}

func TestErrorUnwrapping(t *testing.T) {
	t.Run("Standard Unwrap check", func(t *testing.T) {
		err := fmt.Errorf("context error: %w", ErrSecurityViolation)
		if !errors.Is(err, ErrSecurityViolation) {
			t.Error("failed to unwrap security violation error using errors.Is")
		}
	})

	t.Run("Deep Nesting", func(t *testing.T) {
		err := fmt.Errorf("layer 2: %w", fmt.Errorf("layer 1: %w", ErrRobotsDisallowed))
		if !errors.Is(err, ErrRobotsDisallowed) {
			t.Error("failed to unwrap deeply nested error")
		}
	})

	t.Run("Parallel Error Unwrapping", func(t *testing.T) {
		err := fmt.Errorf("processor failed: %w", ErrDelayRequired)
		if !errors.Is(err, ErrDelayRequired) {
			t.Error("failed to identify retryable error inside processor wrapper")
		}
	})
}

func TestErrorMessages(t *testing.T) {
	if ErrRobotsDisallowed.Error() != "robots.txt disallows crawling" {
		t.Error("ErrRobotsDisallowed message mismatch")
	}
	if ErrSecurityViolation.Error() != "security policy violation: potential prompt injection" {
		t.Error("ErrSecurityViolation message mismatch")
	}
}
