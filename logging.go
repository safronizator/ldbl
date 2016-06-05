package ldbl

import (
	"log"
)

// Optional logger logs only if logger is set.
type OptionalLogger struct {
	LogPrefix string
	logger    *log.Logger
}

func (l *OptionalLogger) SetLogger(logger *log.Logger) {
	l.logger = logger
}

func (l *OptionalLogger) Log(format string, args ...interface{}) {
	if l.logger != nil {
		l.logger.Printf(l.LogPrefix+": "+format, args...)
	}
}
