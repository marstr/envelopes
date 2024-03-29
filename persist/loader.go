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
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/marstr/envelopes"

	"github.com/marstr/collection/v2"
)

// ErrObjectNotFound indicates that a non-existent object was requested.
type ErrObjectNotFound envelopes.ID

func (err ErrObjectNotFound) Error() string {
	return fmt.Sprintf("not able to find object %s", envelopes.ID(err).String())
}

// Loader can instantiate core envelopes objects given just an ID.
type Loader interface {
	LoadTransaction(ctx context.Context, id envelopes.ID, destination *envelopes.Transaction) error
	LoadState(ctx context.Context, id envelopes.ID, destination *envelopes.State) error
	LoadBudget(ctx context.Context, id envelopes.ID, destination *envelopes.Budget) error
	LoadAccounts(ctx context.Context, id envelopes.ID, destination *envelopes.Accounts) error
}

type ErrNoCommonAncestor []envelopes.ID

func (err ErrNoCommonAncestor) Error() string {
	out := &bytes.Buffer{}
	_, _ = fmt.Fprint(out, "no common ancestor found for: ")
	for i := 0; i < len(err); i++ {
		if i >= 6 {
			_, _ = fmt.Fprint(out, ", ...")
			break
		} else {
			_, _ = fmt.Fprintf(out, "%s", err[i])
		}

		if i < len(err)-1 {
			_, _ = fmt.Fprint(out, ", ")
		}
	}

	return out.String()
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

// LoadAncestor reads in and unmarshals a sequence of Transactions, until the main-line ancestor of the given number of
// jumps is loaded.
//
// Note: Calling LoadAncestor with jumps=0 is equivalent to calling Loader.Load with a transaction, but is a hair slower.
func LoadAncestor(ctx context.Context, loader Loader, transaction envelopes.ID, jumps uint) (*envelopes.Transaction, error) {
	var result envelopes.Transaction
	for i := uint(0); i <= jumps; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Intentionally Left Blank
		}
		if err := loader.LoadTransaction(ctx, transaction, &result); err != nil {
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

// LoadImpact finds the change to an envelopes.State associated with an envelopes.Transaction. When transaction has
// only a single parent, the impact is trivial: one just subtracts
func LoadImpact(ctx context.Context, loader Loader, transaction envelopes.Transaction) (envelopes.Impact, error) {
	var err error
	var nca envelopes.Transaction
	var ncaId envelopes.ID

	ncaId, err = NearestCommonAncestorMany(ctx, loader, transaction.Parents)
	if err == nil {
		err = loader.LoadTransaction(ctx, ncaId, &nca)
		if err != nil {
			return envelopes.Impact{}, err
		}
	} else if _, ok := err.(ErrNoCommonAncestor); ok {
		nca.State = &envelopes.State{}
	} else {
		return envelopes.Impact{}, err
	}

	totalDelta := transaction.State.Subtract(*nca.State)

	for _, pid := range transaction.Parents {
		var parent envelopes.Transaction
		err = loader.LoadTransaction(ctx, pid, &parent)
		if err != nil {
			return envelopes.Impact{}, err
		}

		parentDelta := parent.State.Subtract(*nca.State)
		totalDelta = envelopes.State(totalDelta).Subtract(envelopes.State(parentDelta))
	}

	return totalDelta, nil
}

// NearestCommonAncestorMany will find the NearestCommonAncestor for a collection of one or more Transactions.
func NearestCommonAncestorMany(ctx context.Context, loader Loader, heads []envelopes.ID) (envelopes.ID, error) {
	if len(heads) == 0 {
		return envelopes.ID{}, ErrNoCommonAncestor(heads)
	}

	if len(heads) == 1 {
		return heads[0], nil
	}

	var err error
	current := heads[0]
	for i := 1; i < len(heads); i++ {
		select {
		case <-ctx.Done():
			return envelopes.ID{}, ctx.Err()
		default:
			// Intentionally Left Blank
		}

		current, err = NearestCommonAncestor(ctx, loader, current, heads[i])
		if _, ok := err.(ErrNoCommonAncestor); ok {
			return envelopes.ID{}, ErrNoCommonAncestor(heads)
		}
		if err != nil {
			return envelopes.ID{}, err
		}
	}
	return current, nil
}

// NearestCommonAncestor walks the graph created by looking at the Parents of each envelopes.Transaction, finding the
// nearest Transaction that is the ancestor of transactions with both head1 and head2 IDs.
func NearestCommonAncestor(ctx context.Context, loader Loader, head1, head2 envelopes.ID) (envelopes.ID, error) {
	seenLeft := make(map[envelopes.ID]struct{})
	seenRight := make(map[envelopes.ID]struct{})
	toProcessLeft := collection.NewQueue[envelopes.ID](head1)
	toProcessRight := collection.NewQueue[envelopes.ID](head2)

	for !(toProcessLeft.IsEmpty() && toProcessRight.IsEmpty()) {
		select {
		case <-ctx.Done():
			return envelopes.ID{}, ctx.Err()
		default:
			// Intentionally Left Blank
		}

		var current envelopes.Transaction

		if !toProcessLeft.IsEmpty() {
			left, _ := toProcessLeft.Next()
			seenLeft[left] = struct{}{}

			if _, ok := seenRight[left]; ok {
				return left, nil
			}

			if err := loader.LoadTransaction(ctx, left, &current); err != nil {
				return envelopes.ID{}, err
			}

			for _, p := range current.Parents {
				toProcessLeft.Add(p)
			}
		}

		if !toProcessRight.IsEmpty() {

			right, _ := toProcessRight.Next()
			seenRight[right] = struct{}{}

			if _, ok := seenLeft[right]; ok {
				return right, nil
			}

			if err := loader.LoadTransaction(ctx, right, &current); err != nil {
				return envelopes.ID{}, err
			}

			for _, p := range current.Parents {
				toProcessRight.Add(p)
			}
		}
	}

	return envelopes.ID{}, ErrNoCommonAncestor{head1, head2}
}
