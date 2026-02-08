package core

import (
	"errors"
	"time"
)

var (
	ErrSourceExhausted   = errors.New("source stream exhausted")
	ErrProcessorFailed   = errors.New("processing step failed")
	ErrSinkWriteFailed   = errors.New("validation failed")
	ErrMissingNode       = errors.New("node is missing")
	ErrRateLimit         = errors.New("rate limit exceeded")
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
	var re *RetryableError
	if errors.As(err, &re) {
		return true, re.RetryAfter
	}
	return false, 0
}
