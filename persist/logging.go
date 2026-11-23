package persist

import (
	"context"
	"log/slog"

	"github.com/marstr/envelopes"
)

func diagnosticLogger(repository interface{}) *slog.Logger {
	if provider, ok := repository.(envelopes.DiagnosticLoggerProvider); ok {
		return provider.DiagnosticLogger()
	}
	return nil
}

func logDiagnostic(ctx context.Context, logger *slog.Logger, operation, stage string, err error, attrs ...slog.Attr) {
	if logger == nil || !logger.Enabled(ctx, slog.LevelDebug) {
		return
	}
	attrs = append(attrs, slog.String("operation", operation), slog.String("stage", stage))
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
	}
	logger.LogAttrs(ctx, slog.LevelDebug, "persistence operation", attrs...)
}
