package core

import (
	"errors"
	"time"
)

var (
	ErrRobotsDisallowed  = errors.New("robots.txt disallows crawling")
	ErrSecurityViolation = errors.New("security policy violation: potential prompt injection")
	ErrQuotaExceeded     = errors.New("domain crawl quota exceeded")
	ErrDelayRequired     = errors.New("politeness delay required")
)

type RetryableError struct {
	Err        error
	RetryAfter time.Duration
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func IsRetryable(err error) (bool, time.Duration) {
	if err == nil {
		return false, 0
	}

	var re *RetryableError
	if errors.As(err, &re) {
		return true, re.RetryAfter
	}

	if errors.Is(err, ErrDelayRequired) {
		return true, 5 * time.Second
	}

	if errors.Is(err, ErrQuotaExceeded) || errors.Is(err, ErrRobotsDisallowed) || errors.Is(err, ErrSecurityViolation) {
		return false, 0
	}

	return true, 0
}
