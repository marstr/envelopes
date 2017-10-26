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

// Package envelopes provides the basic types and functionality to effectively
// model transactions using the Envelope System of budgeting. Traditionally,
// this is a fairly conservative way of allocating money. It prevents debt
// by limiting individual categories of spending to cash that is physically
// stored in envelopes. If a transaction is larger than what is available in
// an envelope, one would have to explicitly choose which other envelope they
// were going to raid for funds.
//
// However, leaving aside the personal decisions and fundamental philosophy
// about debt, there is still a huge amount of value to thinking of spending
// in a categorical way.
package envelopes

import (
	"bytes"
	"fmt"
)

// Budget encapsulates a single category of spending.
//
// Unlike some budgeting systems, these budgets are based on a debit/credit
// system. There is no built in concept of a monthly reset. However, you can
// build that yourself easily, ever by building a `distribution` which zeroes
// out the balances of all non-rollover budgets and places it in a designated
// savings budget.
type Budget struct {
	Name     string   `json:"name"`
	Balance  int64    `json:"balance"`
	Children []Budget `json:"children,omitempty"`
}

// RecursiveBalance finds the balance of a `Budget` and all of its children.
func (b Budget) RecursiveBalance() (sum int64) {
	sum = b.Balance
	for _, child := range b.Children {
		sum += child.RecursiveBalance()
	}
	return
}

func (b Budget) String() string {
	builder := new(bytes.Buffer)

	// In order to use a lambda recursively, a symbol must be defined for it
	// externally.
	var helper func(Budget)

	// Because `String` shouldn't really be a performance throttling operation,
	// it is more readable to recursively examine the children. Having
	// `builder` declared externally allows a closure to capture it, meaning
	// that we can allocate only one buffer instead of stitching together many
	// strings. Saving the expensive string allocations, and the resulting GC
	// hits, should help mitigate any performance concerns associated with the
	// recursive nature of this function.
	helper = func(b Budget) {
		fmt.Fprintf(builder, "{%q:$%0.2f", b.Name, float64(b.Balance)/100)

		if len(b.Children) > 0 {
			fmt.Fprint(builder, " [")
			for _, child := range b.Children {
				helper(child)
				fmt.Fprint(builder, ", ")
			}
			builder.Truncate(builder.Len() - 2)
			fmt.Fprint(builder, "]")
		}
		fmt.Fprint(builder, "}")
	}

	helper(b)

	return builder.String()
}
