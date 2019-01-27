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
}

type ErrUnloadableType string

func (err ErrUnloadableType) Error() string {
	return fmt.Sprintf("could not load type %q", string(err))
}

// NewErrUnloadableType indicates that a persist.Loader was unable to identify the given object as something it knows
// how to Load.
func NewErrUnloadableType(subject interface{}) ErrUnloadableType {
	return ErrUnloadableType(reflect.TypeOf(subject).Name())
}

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
	err = dl.Load(ctx, unmarshaled.State, &state)
	if err != nil {
		return err
	}

	toLoad.State = &state
	toLoad.Comment = unmarshaled.Comment
	toLoad.Merchant = unmarshaled.Merchant
	toLoad.Time = unmarshaled.Time
	toLoad.Parent = unmarshaled.Parent
	toLoad.Amount = unmarshaled.Amount

	return nil
}

func (dl DefaultLoader) loadState(ctx context.Context, marshaled []byte, toLoad *envelopes.State) error {
	var unmarshaled State
	err := json.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		return err
	}

	var budget envelopes.Budget
	err = dl.Load(ctx, unmarshaled.Budget, &budget)
	if err != nil {
		return err
	}

	err = dl.Load(ctx, unmarshaled.Accounts, &toLoad.Accounts)
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

	toLoad.Balance = unmarshaled.Balance
	toLoad.Children = make(map[string]*envelopes.Budget, len(unmarshaled.Children))
	for name, childID := range unmarshaled.Children {
		var child envelopes.Budget
		err = dl.Load(ctx, childID, &child)
		if err != nil {
			return err
		}
		toLoad.Children[name] = &child
	}

	return nil
}
