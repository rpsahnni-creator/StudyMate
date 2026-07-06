package notifications

import "errors"

// RetryableError marks an error that should be retried with backoff.
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	if e == nil || e.Err == nil {
		return "retryable error"
	}
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// IsRetryable reports whether err (or its chain) is a RetryableError.
func IsRetryable(err error) bool {
	var re *RetryableError
	return errors.As(err, &re)
}
