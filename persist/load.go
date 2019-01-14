// Copyright 2017 Martin Strobel
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package persist

import (
	"context"
	"encoding/json"

	"github.com/marstr/envelopes"
)

// Loader can instantiate core envelopes objects given just an ID.
type Loader interface {
	LoadAccounts(context.Context, envelopes.ID) (envelopes.Accounts, error)
	LoadBudget(context.Context, envelopes.ID) (envelopes.Budget, error)
	LoadState(context.Context, envelopes.ID) (envelopes.State, error)
	LoadTransaction(context.Context, envelopes.ID) (envelopes.Transaction, error)
}

// DefaultLoader wraps a Fetcher and does just the unmarshaling portion.
type DefaultLoader struct {
	Fetcher
}

func (dl DefaultLoader) load(ctx context.Context, id envelopes.ID, target interface{}) (err error) {
	contents, err := dl.Fetch(ctx, id)
	if err != nil {
		return
	}

	err = json.Unmarshal(contents, target)
	return
}

func LoadAll(ctx context.Context, loader Loader, id envelopes.ID) (envelopes.Transaction, envelopes.State, envelopes.Accounts, envelopes.Budget, error) {
	loadedTransaction, err := loader.LoadTransaction(ctx, id)
	if err != nil {
		return envelopes.Transaction{}, envelopes.State{}, envelopes.Accounts{}, envelopes.Budget{}, err
	}

	loadedState, err := loader.LoadState(ctx, loadedTransaction.State())
	if err != nil {
		return envelopes.Transaction{}, envelopes.State{}, envelopes.Accounts{}, envelopes.Budget{}, err
	}

	loadedAccounts, err := loader.LoadAccounts(ctx, loadedState.Accounts())
	if err != nil {
		return envelopes.Transaction{}, envelopes.State{}, envelopes.Accounts{}, envelopes.Budget{}, err
	}

	loadedBudget, err := loader.LoadBudget(ctx, loadedState.Budget())
	if err != nil {
		return envelopes.Transaction{}, envelopes.State{}, envelopes.Accounts{}, envelopes.Budget{}, err
	}

	return loadedTransaction, loadedState, loadedAccounts, loadedBudget, nil
}

func (dl DefaultLoader) LoadAll(ctx context.Context, id envelopes.ID) (envelopes.Transaction, envelopes.State, envelopes.Accounts, envelopes.Budget, error) {
	return LoadAll(ctx, dl, id)
}

// LoadAccounts fetches a list of Accounts in its marshaled form, then unmarshals it into an Accounts object.
func (dl DefaultLoader) LoadAccounts(ctx context.Context, id envelopes.ID) (loaded envelopes.Accounts, err error) {
	err = dl.load(ctx, id, &loaded)
	return
}

// LoadBudget fetches a Budget in its marshaled form, then unmarshals it into a Budget object.
func (dl DefaultLoader) LoadBudget(ctx context.Context, id envelopes.ID) (loaded envelopes.Budget, err error) {
	err = dl.load(ctx, id, &loaded)
	return
}

// LoadState fetches a State in its marshaled form, then unmarshals it into a State object.
func (dl DefaultLoader) LoadState(ctx context.Context, id envelopes.ID) (loaded envelopes.State, err error) {
	err = dl.load(ctx, id, &loaded)
	return
}

// LoadTransaction fetches a Transaction in its marshaled form, then unmarshals it into a Transaction object.
func (dl DefaultLoader) LoadTransaction(ctx context.Context, id envelopes.ID) (loaded envelopes.Transaction, err error) {
	err = dl.load(ctx, id, &loaded)
	return
}
