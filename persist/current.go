package persist

import (
	"context"

	"github.com/marstr/envelopes"
)

// CurrentReader allows the caller to view the state of the CurrentPointer.
type CurrentReader interface {
	// Current finds the ID of the most recently checked out Transaction.
	Current(ctx context.Context) (envelopes.ID, error)

	// CurrentRef finds an identifier for the most recent Transaction that has been committed. Most times, this is the
	// name of the current branch. A repository could also be in HEADless mode, in which case this will return a
	// Transaction ID instead.
	CurrentRef(ctx context.Context) (RefSpec, error)
}

// CurrentWriter allows the caller to modify the state of the Current pointer.
type CurrentWriter interface {
	// WriteCurrent updates the most recently committed transaction. If the repository is in HEADless mode, this will
	// just update the current Transaction ID. If the repository has a branch checked out, this will update the commit
	// that the branch points to.
	WriteCurrent(ctx context.Context, current envelopes.Transaction) error

	// SetCurrent updates which Transaction is currently said to be checked out. It literally replaces the current
	// pointer, and should be used when switching branches or moving between transactions in HEADless mode.
	SetCurrent(ctx context.Context, current RefSpec) error
}
