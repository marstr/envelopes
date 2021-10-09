package json

import (
	"context"
	"encoding/json"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

type Loader=LoaderV2

// LoaderV2 wraps a Fetcher and does just the unmarshaling portion.
type LoaderV2 struct {
	persist.Fetcher

	// Loopback will be called when retrieving sub-object. i.e. It will be invoked when a TransactionV2 needs a StateV2.
	// If it is not set, LoaderV2 will use itself.
	Loopback persist.Loader
}

func (dl LoaderV2) loopback() persist.Loader {
	if dl.Loopback == nil {
		return dl
	}
	return dl.Loopback
}

// Load fetches and parses all objects necessary to fully rehydrate `destination` from wherever it was stashed.
//
// See Also:
// - WriterV2.Write
func (dl LoaderV2) Load(ctx context.Context, id envelopes.ID, destination envelopes.IDer) error {
	// In recursive methods, it is easy to detect that a context has been cancelled between calls to itself.
	// Must have default clause to prevent blocking.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Intentionally Left Blank
	}

	contents, err := dl.Fetch(ctx, id)
	if err != nil {
		return err
	}

	switch destination.(type) {
	case *envelopes.Transaction:
		return dl.loadTransaction(ctx, contents, destination.(*envelopes.Transaction))
	case *envelopes.Budget:
		return dl.loadBudget(ctx, contents, destination.(*envelopes.Budget))
	case *envelopes.State:
		return dl.loadState(ctx, contents, destination.(*envelopes.State))
	case *envelopes.Accounts:
		return json.Unmarshal(contents, destination)
	default:
		return persist.NewErrUnloadableType(destination)
	}
}

func (dl LoaderV2) loadTransaction(ctx context.Context, marshaled []byte, toLoad *envelopes.Transaction) error {
	var unmarshaled TransactionV2
	err := json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		return err
	}

	var state envelopes.State
	err = dl.loopback().Load(ctx, unmarshaled.State, &state)
	if err != nil {
		return err
	}

	toLoad.State = &state
	toLoad.Comment = unmarshaled.Comment
	toLoad.Merchant = unmarshaled.Merchant
	toLoad.ActualTime = unmarshaled.ActualTime
	toLoad.EnteredTime = unmarshaled.EnteredTime
	toLoad.PostedTime = unmarshaled.PostedTime
	toLoad.Parents = unmarshaled.Parent
	toLoad.Amount = envelopes.Balance(unmarshaled.Amount)
	toLoad.Committer.FullName = unmarshaled.Committer.FullName
	toLoad.Committer.Email = unmarshaled.Committer.Email
	toLoad.RecordID = envelopes.BankRecordID(unmarshaled.RecordId)

	return nil
}

func (dl LoaderV2) loadState(ctx context.Context, marshaled []byte, toLoad *envelopes.State) error {
	var unmarshaled StateV2
	err := json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		return err
	}

	var budget envelopes.Budget
	err = dl.loopback().Load(ctx, unmarshaled.Budget, &budget)
	if err != nil {
		return err
	}

	err = dl.loopback().Load(ctx, unmarshaled.Accounts, &toLoad.Accounts)
	if err != nil {
		return err
	}

	toLoad.Budget = &budget
	return nil
}

func (dl LoaderV2) loadBudget(ctx context.Context, marshaled []byte, toLoad *envelopes.Budget) error {
	var unmarshaled BudgetV2
	err := json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		return err
	}

	toLoad.Balance = envelopes.Balance(unmarshaled.Balance)
	toLoad.Children = make(map[string]*envelopes.Budget, len(unmarshaled.Children))
	for name, childID := range unmarshaled.Children {
		var child envelopes.Budget
		err = dl.loopback().Load(ctx, childID, &child)
		if err != nil {
			return err
		}
		toLoad.Children[name] = &child
	}

	return nil
}

