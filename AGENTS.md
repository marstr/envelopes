# Contributor guidance for coding agents

- Keep the README focused on people using the library. Put detailed subsystem
  documentation under `docs/` and link to it from here.
- Logging is opt-in through an application-provided `*slog.Logger`; never fall back
  to `slog.Default()`. Keep diagnostics out of stateless arithmetic and hot paths.
- Log returned failures at Debug, reserving Warn for failures the API cannot return.
  Do not log financial payloads, bank-record IDs, or filenames encoding those IDs.
- See [logging documentation](docs/logging.md) for logger setup, the optional
  `envelopes.DiagnosticLoggerProvider` capability, and diagnostic semantics.
