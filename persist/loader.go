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
	"errors"
	"github.com/marstr/envelopes"
)

// Loader can instantiate core envelopes objects given just an ID.
type Loader interface {
	Load(ctx context.Context, id envelopes.ID, destination envelopes.IDer) error
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