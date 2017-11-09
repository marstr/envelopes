package persist

import (
	"github.com/marstr/envelopes"
)

type Writer interface {
	WriteBudget(envelopes.Budget) error
	WriteState(envelopes.State) error
	WriteTransaction(envelopes.Transaction) error
}
