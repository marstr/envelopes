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

	"github.com/marstr/envelopes"
)

// Writer defines a contract that allows an object to express that it knows how to persist
// an object so that it can be recalled using an instance of an object that satisfies `persist.Fetch`.
type Writer interface {
	WriteAccounts(context.Context, envelopes.Accounts) error
	// WriteBudget must persist a budget in a means that it is fetchable using a `persist.Fetch`.
	WriteBudget(context.Context, envelopes.Budget) error
	// WriteCurrent stores the ID of the Transaction that should be considered the most up-to-date one.
	WriteCurrent(context.Context, envelopes.Transaction) error
	// WriteState must persist a budget in a means that it is fetchable using a `persist.Fetch`.
	WriteState(context.Context, envelopes.State) error
	// WriteTransaction must persist a budget in a means that it is fetchable using a `persist.Fetch`.
	WriteTransaction(context.Context, envelopes.Transaction) error
}

func WriteAll(ctx context.Context, writer Writer, t envelopes.Transaction, s envelopes.State, a envelopes.Accounts, b envelopes.Budget) (err error) {
	encounteredErrs := new(bytes.Buffer)
	if got := a.ID(); !s.Accounts().Equal(got) {
		fmt.Fprintf(encounteredErrs, "expected Account to have ID %q\n", got)
	}

	if got := b.ID(); !s.Budget().Equal(got) {
		fmt.Fprintf(encounteredErrs, "expected Budget to have ID %q\n", got)
	}

	if got := s.ID(); !t.State().Equal(got) {
		fmt.Fprintf(encounteredErrs, "expected State to have ID %q\n", got)
	}

	if encounteredErrs.Len() > 0 {
		err = errors.New(encounteredErrs.String())
		return
	}

	err = writer.WriteAccounts(ctx, a)
	if err != nil {
		return
	}

	err = writer.WriteBudget(ctx, b)
	if err != nil {
		return
	}

	err = writer.WriteState(ctx, s)
	if err != nil {
		return
	}

	err = writer.WriteTransaction(ctx, t)
	return
}
