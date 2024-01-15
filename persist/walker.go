// "Money talks, bullshit walks" -Unknown

package persist

import (
	"context"

	"github.com/marstr/collection/v2"
	"github.com/marstr/envelopes"
)

// WalkFunc will be called by a Walker as it encounters transactions.
type WalkFunc func(ctx context.Context, id envelopes.ID, transaction envelopes.Transaction) error

// ErrSkipAncestors allows a WalkFunc to communicate that parents of this transaction shouldn't be visited.
// If other Transactions have shared parent, but don't return ErrSkipAncestors, the shared parents will still
// be visited.
type ErrSkipAncestors struct{}

func (err ErrSkipAncestors) Error() string {
	return "Don't add this Transaction's parents to the list to be processed."
}

type Walker struct {
	// Loader fetches objects for further processing. This must be populated for Walker to function properly.
	Loader Loader

	// MaxDepth controls how many generations beyond the provided heads this Walker will process. If its value is '0',
	// no restrictions are placed, and it will walk the entire envelopes.Transaction history.
	MaxDepth uint
}

func (w *Walker) Walk(ctx context.Context, action WalkFunc, heads ...envelopes.ID) error {
	type toProcessEntry struct {
		envelopes.ID
		Depth uint
	}

	processed := make(map[envelopes.ID]struct{})
	toProcess := collection.NewLinkedList[toProcessEntry]()

	for i := range heads {
		toProcess.AddBack(toProcessEntry{
			ID:    heads[i],
			Depth: 0,
		})
	}

	for collection.Any[toProcessEntry](toProcess) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Intentionally Left Blank
		}

		currentEntry, _ := toProcess.RemoveFront()

		if _, seen := processed[currentEntry.ID]; seen || (w.MaxDepth > 0 && currentEntry.Depth > w.MaxDepth) {
			continue
		}

		var current envelopes.Transaction
		err := w.Loader.LoadTransaction(ctx, currentEntry.ID, &current)
		if err != nil {
			return err
		}
		processed[currentEntry.ID] = struct{}{}

		err = action(ctx, currentEntry.ID, current)
		if err != nil {
			switch err.(type) {
			case ErrSkipAncestors:
				continue
			default:
				return err
			}
		}

		for i := range current.Parents {
			toProcess.AddBack(toProcessEntry{
				ID:    current.Parents[i],
				Depth: currentEntry.Depth + 1,
			})
		}
	}

	return nil
}
