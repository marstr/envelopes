package persist

import (
	"github.com/marstr/envelopes"
)

// Fetcher can grab the marshaled form of an Object given an ID.
type Fetcher interface {
	Fetch(envelopes.ID) ([]byte, error)
}
