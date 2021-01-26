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
	"errors"
	"fmt"
	"reflect"

	"github.com/marstr/envelopes"
)

// Loader can instantiate core envelopes objects given just an ID.
type Loader interface {
	Load(ctx context.Context, id envelopes.ID, destination envelopes.IDer) error
}

// DefaultLoader wraps a Fetcher and does just the unmarshaling portion.
type DefaultLoader struct {
	Fetcher

	// Loopback will be called when retrieving sub-object. i.e. It will be invoked when a Transaction needs a State.
	// If it is not set, DefaultLoader will use itself.
	Loopback Loader
}

// ErrObjectNotFound indicates that a non-existent object was requested.
type ErrObjectNotFound envelopes.ID

func (err ErrObjectNotFound) Error() string {
	return fmt.Sprintf("not able to find object %s", envelopes.ID(err).String())
}

// ErrUnloadableType indicates that a Loader is unable to recognize the specified type.
type ErrUnloadableType string

func (err ErrUnloadableType) Error() string {
	return fmt.Sprintf("could not load type %q", string(err))
}

// NewErrUnloadableType indicates that a persist.Loader was unable to identify the given object as something it knows
// how to Load.
func NewErrUnloadableType(subject interface{}) ErrUnloadableType {
	return ErrUnloadableType(reflect.TypeOf(subject).Name())
}

func (dl DefaultLoader) loopback() Loader {
	if dl.Loopback == nil {
		return dl
	}
	return dl.Loopback
}

// Load fetches and parses all objects necessary to fully rehydrate `destination` from wherever it was stashed.
//
// See Also:
// - DefaultWriter.Write
func (dl DefaultLoader) Load(ctx context.Context, id envelopes.ID, destination envelopes.IDer) error {
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
		return NewErrUnloadableType(destination)
	}
}

func (dl DefaultLoader) loadTransaction(ctx context.Context, marshaled []byte, toLoad *envelopes.Transaction) error {
	var unmarshaled Transaction
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

func (dl DefaultLoader) loadState(ctx context.Context, marshaled []byte, toLoad *envelopes.State) error {
	var unmarshaled State
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

func (dl DefaultLoader) loadBudget(ctx context.Context, marshaled []byte, toLoad *envelopes.Budget) error {
	var unmarshaled Budget
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

// LoadAncestor reads in and unmarshals a sequence of Transactions, until the main-line ancestor of the given number of
// jumps is loaded.
//
// Note: Calling LoadAncestor with jumps=0 is equivalent to calling Loader.Load with a transaction, but is a hair slower.
func LoadAncestor(ctx context.Context, loader Loader, transaction envelopes.ID, jumps uint) (*envelopes.Transaction, error) {
	var result envelopes.Transaction
	for i := uint(0); i <= jumps; i++ {
		if err := loader.Load(ctx, transaction, &result); err != nil {
			return nil, err
		}
		if len(result.Parents) > 0 {
			transaction = result.Parents[0]
		} else if i < jumps {
			return nil, errors.New("no such ancestor")
		}
	}
	return &result, nil
}
