package json

import (
	"context"
	"encoding/json"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

func NewLoaderV2(fetcher persist.Fetcher) (*LoaderV2, error) {
	retval := &LoaderV2{
		Fetcher: fetcher,
	}
	retval.loopback = retval
	return retval, nil
}

func NewLoaderV2WithLoopback(fetcher persist.Fetcher, loopback persist.Loader) (*LoaderV2, error) {
	retval := &LoaderV2{
		Fetcher:  fetcher,
		loopback: loopback,
	}
	return retval, nil
}

// LoaderV2 wraps a Fetcher and does just the unmarshaling portion.
type LoaderV2 struct {
	persist.Fetcher

	// Loopback will be called when retrieving sub-object. i.e. It will be invoked when a TransactionV2 needs a StateV2.
	// If it is not set, LoaderV2 will use itself.
	loopback persist.Loader
}

func (dl LoaderV2) LoadTransaction(ctx context.Context, id envelopes.ID, toLoad *envelopes.Transaction) error {
	marshaled, err := dl.Fetch(ctx, id)
	if err != nil {
		return err
	}

	var unmarshaled TransactionV2
	err = json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		return err
	}

	var state envelopes.State
	err = dl.loopback.LoadState(ctx, unmarshaled.State, &state)
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

func (dl LoaderV2) LoadState(ctx context.Context, id envelopes.ID, toLoad *envelopes.State) error {
	marshaled, err := dl.Fetch(ctx, id)
	if err != nil {
		return err
	}

	var unmarshaled StateV2
	err = json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		return err
	}

	var budget envelopes.Budget
	err = dl.loopback.LoadBudget(ctx, unmarshaled.Budget, &budget)
	if err != nil {
		return err
	}

	err = dl.loopback.LoadAccounts(ctx, unmarshaled.Accounts, &toLoad.Accounts)
	if err != nil {
		return err
	}

	toLoad.Budget = &budget
	return nil
}

func (dl LoaderV2) LoadBudget(ctx context.Context, id envelopes.ID, toLoad *envelopes.Budget) error {
	marshaled, err := dl.Fetch(ctx, id)
	if err != nil {
		return err
	}

	var unmarshaled BudgetV2
	err = json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		return err
	}

	toLoad.Balance = envelopes.Balance(unmarshaled.Balance)
	toLoad.Children = make(map[string]*envelopes.Budget, len(unmarshaled.Children))
	for name, childID := range unmarshaled.Children {
		var child envelopes.Budget
		err = dl.loopback.LoadBudget(ctx, childID, &child)
		if err != nil {
			return err
		}
		toLoad.Children[name] = &child
	}

	return nil
}

func (dl LoaderV2) LoadAccounts(ctx context.Context, id envelopes.ID, toLoad *envelopes.Accounts) error {
	marshaled, err := dl.Fetch(ctx, id)
	if err != nil {
		return err
	}

	return json.Unmarshal(marshaled, toLoad)
}
