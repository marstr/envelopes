# Persistence diagnostics

Filesystem persistence is silent by default, including when the executable configures
`slog.Default()`. Supply your application's `*slog.Logger` to opt in:

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})).With("component", "envelopes")

repo, err := filesystem.OpenRepository(ctx, root,
    filesystem.RepositoryLogger(logger),
)
```

The same option works with `OpenRepositoryWithCache`. For a standalone `FileSystem`
or `FilesystemBankRecordIDIndex`, set its `Logger` field in a keyed struct literal.
Configure it before concurrent use; nil disables logging. The application owns the
handler, destination, level, and any additional attributes or redaction.

Debug records describe filesystem I/O, reference operations, repository configuration,
bank-index operations, and commit/clone progress. Returned failures are also Debug: the executable should
handle and report its final error rather than receive duplicate Error logs from each
library layer. Warn records report failures that existing APIs do not return, such as
ignored close errors or asynchronous index read/parse failures. Arithmetic, cache hits,
JSON codecs, and per-node graph traversal do not emit their own logs.

`persist.Commit` and `persist.BareClone` reuse a repository's logger through the
optional `envelopes.DiagnosticLoggerProvider` capability; custom repositories can
implement it without changing the existing repository interfaces. The capability
lives in the root package so other packages can use it without depending on `persist`;
the persistence-specific logging helpers remain private to `persist`.
Clone prefers the destination's logger, falling back to the source's. Progress records distinguish
written objects from completed reference updates; clone summaries count successfully
written branches, tags, and transactions without logging every visited object.
`persist.Resolve` and `persist.BareResolve` use the same optional capability to
record the successful interpretation (transaction, branch, tag, HEAD, or ancestor
expression) and failed branch/tag probes at Debug level. Resolution records include
reference names and expressions but not the referenced objects' contents.
JSON loaders add the failing object type, ID, and JSON format version to decoding
errors without adding logs or including the serialized payload in the new context.
The underlying error remains available through `errors.Is` and `errors.As`;
fetch failures are returned unchanged.

Debug logs can contain non-index filesystem paths and object IDs. They do not include
financial payloads, comments, user details, attachment contents, bank-record IDs, or
bank-index filenames (which reversibly encode those IDs). Treat debug logs as
potentially sensitive and choose their destination appropriately.

Repository options are applied after configuration detection, so failures before
logger injection remain returned errors without library logs. Stream-open records
do not mean contents have been consumed, and copy records do not guarantee durable
storage. Directory-listing summaries describe emitted entries; the underlying
enumerator does not expose filesystem errors.

Adding optional logger fields preserves keyed literals but requires updating any
unkeyed `FileSystem` or `FilesystemBankRecordIDIndex` literals.
Repository options now reach the backing object store as well as the returned
filesystem. A new repository saves its selected object layout; changing the layout
of an existing repository is rejected rather than leaving its objects inaccessible.
