package persist

import (
	"context"
	"io"

	"github.com/marstr/envelopes"
)

// Stasher is the inverse of a Fetcher. Instead of being able to retrieve raw bytes associated with a particular IDable
// object, it is able to place them.
type Stasher interface {
	Stash(ctx context.Context, id envelopes.ID, payload []byte) error
	// StashReadCloser takes ownership of payload. Implementations are responsible for reading from and
	// closing the provided io.ReadCloser; callers must not close or reuse payload after calling this method.
	StashReadCloser(ctx context.Context, id envelopes.ID, payload io.ReadCloser) error
}
