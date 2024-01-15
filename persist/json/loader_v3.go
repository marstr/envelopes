package json

import (
	"context"
	"encoding/json"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

type Loader = LoaderV3

func NewLoaderV3(fetcher persist.Fetcher) (*LoaderV3, error) {
	retval := &LoaderV3{
		Fetcher: fetcher,
	}
	retval.loopback = retval
	return retval, nil
}

func NewLoaderV3WithLoopback(fetcher persist.Fetcher, loopback persist.Loader) (*LoaderV3, error) {
	retval := &LoaderV3{
		Fetcher:  fetcher,
		loopback: loopback,
	}
	return retval, nil
}

// LoaderV3 wraps a Fetcher and does just the unmarshaling portion.
type LoaderV3 struct {
	persist.Fetcher

	// Loopback will be called when retrieving sub-object. i.e. It will be invoked when a TransactionV3 needs a StateV3.
	// If it is not set, LoaderV3 will use itself.
	loopback persist.Loader
}

// Load fetches and parses all objects necessary to fully rehydrate `destination` from wherever it was stashed.
//
// See Also:
// - WriterV3.Write
func (dl LoaderV3) Load(ctx context.Context, id envelopes.ID, destination envelopes.IDer) error {
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

func (dl LoaderV3) loadTransaction(ctx context.Context, marshaled []byte, toLoad *envelopes.Transaction) error {
	var unmarshaled TransactionV3
	err := json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		return err
	}

	var state envelopes.State
	err = dl.loopback.Load(ctx, unmarshaled.State, &state)
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

func (dl LoaderV3) loadState(ctx context.Context, marshaled []byte, toLoad *envelopes.State) error {
	var unmarshaled StateV3
	err := json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		return err
	}

	var budget envelopes.Budget
	err = dl.loopback.Load(ctx, unmarshaled.Budget, &budget)
	if err != nil {
		return err
	}

	err = dl.loopback.Load(ctx, unmarshaled.Accounts, &toLoad.Accounts)
	if err != nil {
		return err
	}

	toLoad.Budget = &budget
	return nil
}

func (dl LoaderV3) loadBudget(ctx context.Context, marshaled []byte, toLoad *envelopes.Budget) error {
	var unmarshaled BudgetV3
	err := json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		return err
	}

	toLoad.Balance = envelopes.Balance(unmarshaled.Balance)
	toLoad.Children = make(map[string]*envelopes.Budget, len(unmarshaled.Children))
	for name, childID := range unmarshaled.Children {
		var child envelopes.Budget
		err = dl.loopback.Load(ctx, childID, &child)
		if err != nil {
			return err
		}
		toLoad.Children[name] = &child
	}

	return nil
}
