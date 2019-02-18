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
	"crypto/sha1"
	"fmt"
	"sort"
)

// Budget encapsulates a single category of spending.
//
// Unlike some budgeting systems, these budgets are based on a debit/credit
// system. There is no built in concept of a monthly reset. However, you can
// build that yourself easily, ever by building a `distribution` which zeroes
// out the balances of all non-rollover budgets and places it in a designated
// savings budget.
type Budget struct {
	Balance  Balance
	Children map[string]*Budget
}

func (b Budget) deepCopy() Budget {
	var clone Budget
	clone.Balance = b.Balance
	clone.Children = make(map[string]*Budget, len(b.Children))

	for childName, child := range b.Children {
		clonedChild := child.deepCopy()
		clone.Children[childName] = &clonedChild
	}
	return clone
}

// ID fetches the Unique Identifier associated with this Budget.
func (b Budget) ID() ID {
	marshaled, err := b.MarshalText()
	if err != nil {
		return ID{}
	}
	return sha1.Sum(marshaled)
}

// MarshalText computes a deterministic string that uniquely represents this Budget.
func (b Budget) MarshalText() ([]byte, error) {
	// To make deterministic text, the names must be in a predictable,
	// reproducible order. Therefor, instead of reading directly from the
	// map, we must first extract the names and alphabetize them.
	childNames := make([]string, 0, len(b.Children))
	for childName := range b.Children {
		childNames = append(childNames, childName)
	}
	sort.Strings(childNames)

	// Fetch, clear, and promise to return a buffer to hold the ID defining
	// characteristics of this IDer.
	identityBuilder := identityBuilders.Get().(*bytes.Buffer)
	identityBuilder.Reset()
	defer identityBuilders.Put(identityBuilder)

	_, err := fmt.Fprintf(identityBuilder, "balance %s", b.Balance)
	if err != nil {
		return nil, err
	}

	for _, childName := range childNames {
		childID := b.Children[childName].ID()
		_, err = fmt.Fprintf(identityBuilder, "child %s %x\n", childName, childID)
		if err != nil {
			return nil, err
		}
	}

	return identityBuilder.Bytes(), nil
}

// Equal determines whether or not two instances of Budget share the same balance and
// have children that are all equal and have the same name.
func (b Budget) Equal(other Budget) bool {
	if b.Balance != other.Balance {
		return false
	}

	if len(b.Children) != len(other.Children) {
		return false
	}

	for name, child := range b.Children {
		if otherChild, ok := other.Children[name]; !(ok && child.Equal(*otherChild)) {
			return false
		}
	}
	return true
}

// RecursiveBalance finds the balance of a `Budget` and all of its children.
//
// See Also:
// 	Budget.Balance
func (b Budget) RecursiveBalance() (sum Balance) {
	sum = b.Balance
	for _, child := range b.Children {
		sum += child.RecursiveBalance()
	}
	return
}

// ChildNames returns an alphabetically sorted list of the names of each of the chilren
// of this budget.
func (b Budget) ChildNames() (results []string) {
	results = make([]string, 0, len(b.Children))

	for name := range b.Children {
		results = append(results, name)
	}

	sort.Strings(results)
	return
}

func (b Budget) String() string {
	builder := new(bytes.Buffer)

	// In order to use a lambda recursively, a symbol must be defined for it
	// externally.
	var helper func(string, Budget)

	// Because `String` shouldn't really be a performance throttling operation,
	// it is more readable to recursively examine the children. Having
	// `builder` declared externally allows a closure to capture it, meaning
	// that we can allocate only one buffer instead of stitching together many
	// strings. Saving the expensive string allocations, and the resulting GC
	// hits, should help mitigate any performance concerns associated with the
	// recursive nature of this function.
	helper = func(currentName string, b Budget) {
		currentBalance := float64(b.Balance) / 100
		builder.WriteRune('{')

		if currentName != "" {
			builder.WriteString(currentName)
			builder.WriteRune(':')
		}
		_, err := fmt.Fprintf(builder, "$%0.2f", currentBalance)
		if err != nil {
			return
		}

		if len(b.Children) > 0 {
			const childSuffix = ", "
			builder.WriteString(" [")
			for childName, child := range b.Children {
				helper(childName, *child)
				builder.WriteString(childSuffix)
			}
			builder.Truncate(builder.Len() - len(childSuffix))
			builder.WriteRune(']')
		}
		builder.WriteRune('}')
	}

	helper("", b)

	return builder.String()
}
