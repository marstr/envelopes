package persist

import (
	"context"
	"github.com/marstr/envelopes"
)

// Stasher is the inverse of a Fetcher. Instead of being able to retrieve raw bytes associated with a particular IDable
// object, it is able to place them.
type Stasher interface {
	Stash(ctx context.Context, id envelopes.ID, payload []byte) error
}