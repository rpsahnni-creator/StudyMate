package notifications

import "log"

// StdLogger adapts the standard library logger to the notifications.Logger interface.
type StdLogger struct{}

func (StdLogger) Info(msg string, args ...interface{}) {
	log.Printf("[notifications] INFO %s %v", msg, args)
}

func (StdLogger) Error(msg string, args ...interface{}) {
	log.Printf("[notifications] ERROR %s %v", msg, args)
}

func (StdLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[notifications] WARN %s %v", msg, args)
}

func (StdLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[notifications] DEBUG %s %v", msg, args)
}
