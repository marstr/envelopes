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

	"github.com/devigned/tab"

	"github.com/marstr/envelopes"
)

const defaultWriteroperationPrefix = persistOperationPrefix + ".DefaultWriter"

type (
	// Writer defines a contract that allows an object to express that it knows how to persist
	// an object so that it can be recalled using an instance of an object that satisfies `persist.Fetch`.
	Writer interface {
		// Write persists an object in durable storage where it can be retrieved later.
		Write(ctx context.Context, subject envelopes.IDer) error
	}

	// DefaultWriter knows how to navigate the envelopes object model and stash each individual component of an object.
	DefaultWriter struct {
		Stasher
	}
)

// Write uses the Stasher it is composed of to write the given Envelopes object to a persistent storage space.
func (dw DefaultWriter) Write(ctx context.Context, subject envelopes.IDer) error {
	var span tab.Spanner
	const operationName = defaultWriteroperationPrefix + ".Write"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	// In recursive methods, it is easy to detect that a context has been cancelled between calls to itself.
	// Must have default clause to prevent blocking.
	select {
	case <-ctx.Done():
		err := ctx.Err()
		span.Logger().Error(err)
		return err
	default:
		// Intentionally Left Blank
	}

	span.AddAttributes(tab.StringAttribute("entityType", reflect.TypeOf(subject).Name()))

	switch subject.(type) {
	case envelopes.Transaction:
		return dw.writeTransaction(ctx, subject.(envelopes.Transaction))
	case *envelopes.Transaction:
		return dw.writeTransaction(ctx, *subject.(*envelopes.Transaction))
	case envelopes.State:
		return dw.writeState(ctx, subject.(envelopes.State))
	case *envelopes.State:
		return dw.writeState(ctx, *subject.(*envelopes.State))
	case envelopes.Budget:
		return dw.writeBudget(ctx, subject.(envelopes.Budget))
	case *envelopes.Budget:
		return dw.writeBudget(ctx, *subject.(*envelopes.Budget))
	case envelopes.Accounts:
		return dw.writeAccounts(ctx, subject.(envelopes.Accounts))
	case *envelopes.Accounts:
		return dw.writeAccounts(ctx, *subject.(*envelopes.Accounts))
	default:
		err := fmt.Errorf("unknown type: %s", reflect.TypeOf(subject).Name())
		span.Logger().Error(err)
		return err
	}
}

func (dw DefaultWriter) writeTransaction(ctx context.Context, subject envelopes.Transaction) error {
	var span tab.Spanner
	const operationName = defaultWriteroperationPrefix + ".writeTransaction"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	if subject.State == nil {
		subject.State = &envelopes.State{}
	}

	var toMarshal Transaction
	toMarshal.Amount = subject.Amount
	toMarshal.Parent = subject.Parent
	toMarshal.State = subject.State.ID()
	toMarshal.Comment = subject.Comment
	toMarshal.Merchant = subject.Merchant
	toMarshal.Time = subject.Time

	err := dw.Write(ctx, subject.State)
	if err != nil {
		span.Logger().Error(err)
		return err
	}

	marshaled, err := json.Marshal(toMarshal)
	if err != nil {
		span.Logger().Error(err)
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}

func (dw DefaultWriter) writeState(ctx context.Context, subject envelopes.State) error {
	var span tab.Spanner
	const operationName = defaultWriteroperationPrefix + ".writeState"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	if subject.Accounts == nil {
		subject.Accounts = make(envelopes.Accounts, 0)
	}
	err := dw.Write(ctx, subject.Accounts)
	if err != nil {
		span.Logger().Error(err)
		return err
	}

	if subject.Budget == nil {
		subject.Budget = &envelopes.Budget{}
	}
	err = dw.Write(ctx, subject.Budget)
	if err != nil {
		span.Logger().Error(err)
		return err
	}

	var toMarshal State
	toMarshal.Accounts = subject.Accounts.ID()
	toMarshal.Budget = subject.Budget.ID()

	marshaled, err := json.Marshal(toMarshal)
	if err != nil {
		span.Logger().Error(err)
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}

func (dw DefaultWriter) writeBudget(ctx context.Context, subject envelopes.Budget) error {
	var span tab.Spanner
	const operationName = defaultWriteroperationPrefix + ".writeBudget"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	if subject.Children == nil {
		subject.Children = make(map[string]*envelopes.Budget, 0)
	}
	for _, child := range subject.Children {
		err := dw.Write(ctx, child)
		if err != nil {
			span.Logger().Error(err)
			return err
		}
	}

	var toMarshal Budget
	toMarshal.Balance = subject.Balance
	toMarshal.Children = make(map[string]envelopes.ID, len(subject.Children))
	for name, child := range subject.Children {
		toMarshal.Children[name] = child.ID()
	}

	marshaled, err := json.Marshal(toMarshal)
	if err != nil {
		span.Logger().Error(err)
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}

func (dw DefaultWriter) writeAccounts(ctx context.Context, subject envelopes.Accounts) error {
	var span tab.Spanner
	const operationName = defaultWriteroperationPrefix + ".writeAccounts"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	marshaled, err := json.Marshal(subject)
	if err != nil {
		span.Logger().Error(err)
		return err
	}

	return dw.Stash(ctx, subject.ID(), marshaled)
}
