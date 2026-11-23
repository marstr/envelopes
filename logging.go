package envelopes

import "log/slog"

// DiagnosticLoggerProvider optionally supplies a logger for library diagnostics.
// Implementations without this capability, or returning nil, do not emit diagnostics.
type DiagnosticLoggerProvider interface {
	DiagnosticLogger() *slog.Logger
}
