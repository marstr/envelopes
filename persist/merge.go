package persist

import (
	"context"

	"github.com/marstr/envelopes"
)

type Conflict struct {
}

func Merge(ctx context.Context, repo RepositoryReader, heads []RefSpec) (merged envelopes.State, conflicts []Conflict, err error) {
	var headIDs []envelopes.ID
	headIDs, err = ResolveMany(ctx, repo, heads)
	if err != nil {
		return
	}

	hydratedHeads := make([]envelopes.Transaction, len(headIDs))
	for i, head := range headIDs {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
			// Intentionally Left Blank
		}

		err = repo.LoadTransaction(ctx, head, &hydratedHeads[i])
		if err != nil {
			return
		}
	}

	var ncaID envelopes.ID
	ncaID, err = NearestCommonAncestorMany(ctx, repo, headIDs)
	if err != nil {
		return
	}

	var nca envelopes.Transaction
	err = repo.LoadTransaction(ctx, ncaID, &nca)
	if err != nil {
		return
	}

	merged = nca.State.DeepCopy()

	for i := range hydratedHeads {
		delta := hydratedHeads[i].State.Subtract(*nca.State)
		merged = envelopes.State(merged.Add(envelopes.State(delta)))
	}

	// TODO Walk all transactions from HEADs to nca and find Transactions with duplicate BankIDs, report those as conflicts in return value.

	return
}
