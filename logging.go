package m3da

import (
	"fmt"
	"log/slog"
)

// Internal logging functions used throughout the library
// These use the default slog logger which can be configured by the consuming application
func debugf(format string, args ...interface{}) {
	slog.Debug(fmt.Sprintf(format, args...))
}

func infof(format string, args ...interface{}) {
	slog.Info(fmt.Sprintf(format, args...))
}

func warnf(format string, args ...interface{}) {
	slog.Warn(fmt.Sprintf(format, args...))
}

func errorf(format string, args ...interface{}) {
	slog.Error(fmt.Sprintf(format, args...))
}
