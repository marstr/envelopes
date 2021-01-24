// "Money talks, bullshit walks" -Unknown

package traverse

import (
	"context"
	"github.com/marstr/collection"
	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

type WalkFunc func(ctx context.Context, transaction envelopes.Transaction) error

type Walker struct {
	Loader persist.Loader
}

func (w *Walker) Walk(ctx context.Context, action WalkFunc, heads ...envelopes.ID) error {
	processed := make(map[envelopes.ID]struct{})
	toProcess := collection.NewLinkedList()
	for i := range heads {
		toProcess.AddBack(heads[i])
	}

	for collection.Any(toProcess) {
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

		err = action(ctx, current)
		if err != nil {
			return err
		}

		for i := range current.Parent {
			toProcess.AddBack(current.Parent[i])
		}

		processed[currentId] = struct{}{}
	}

	return nil
}
