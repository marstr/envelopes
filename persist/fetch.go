package persist

import (
	"io"

	"github.com/marstr/envelopes"
)

// Fetcher can grab the marshaled form of an Object given an ID.
type Fetcher interface {
	Fetch(envelopes.ID) (io.Reader, error)
}
