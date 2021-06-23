// "Money talks, bullshit walks" -Unknown

package persist

import (
	"context"
	"github.com/marstr/collection"
	"github.com/marstr/envelopes"
)

type WalkFunc func(ctx context.Context, transaction envelopes.Transaction) error

// ErrSkipAncestors allows a WalkFunc to communicate that parents of this transaction shouldn't be visited.
// If other Transactions have shared parent, but don't return ErrSkipAncestors, the shared parents will still
// be visited.
type ErrSkipAncestors struct{}

func (err ErrSkipAncestors) Error() string {
	return "Don't add this Transaction's parents to the list to be processed."
}

type Walker struct {
	Loader Loader
}

func (w *Walker) Walk(ctx context.Context, action WalkFunc, heads ...envelopes.ID) error {
	processed := make(map[envelopes.ID]struct{})
	toProcess := collection.NewLinkedList()
	for i := range heads {
		toProcess.AddBack(heads[i])
	}

	for collection.Any(toProcess) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Intentionally Left Blank
		}

		frontNode, _ := toProcess.RemoveFront()
		currentId := frontNode.(envelopes.ID)

		if _, seen := processed[currentId]; seen {
			continue
		}

		var current envelopes.Transaction
		err := w.Loader.Load(ctx, currentId, &current)
		if err != nil {
			return err
		}
		processed[currentId] = struct{}{}

		err = action(ctx, current)
		if err != nil {
			switch err.(type) {
			case ErrSkipAncestors:
				continue
			default:
				return err
			}
		}

		for i := range current.Parents {
			toProcess.AddBack(current.Parents[i])
		}
	}

	return nil
}
