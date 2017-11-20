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
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// Budget encapsulates a single category of spending.
//
// Unlike some budgeting systems, these budgets are based on a debit/credit
// system. There is no built in concept of a monthly reset. However, you can
// build that yourself easily, ever by building a `distribution` which zeroes
// out the balances of all non-rollover budgets and places it in a designated
// savings budget.
type Budget struct {
	balance  int64
	children map[string]Budget
}

func (b Budget) deepCopy() (updated Budget) {
	updated.balance = b.balance
	if len(b.children) > 0 {
		updated.children = make(map[string]Budget, len(b.children))
		for name, child := range b.children {
			updated.children[name] = child
		}
	}
	return
}

// ID fetches the Unique Identifier associated with this Budget.
func (b Budget) ID() (id ID) {
	id, _ = NewID(b)
	return
}

// Equal determines whether or not two instances of Budget share the same balance and
// have children that are all equal and have the same name.
func (b Budget) Equal(other Budget) bool {
	if b.Balance() != other.Balance() {
		return false
	}

	if len(b.children) != len(other.children) {
		return false
	}

	for name, child := range b.children {
		if otherChild, ok := other.children[name]; !(ok && child.Equal(otherChild)) {
			return false
		}
	}
	return true
}

// Balance retrieves the amount of funds directly available in this `Budget
// but none of its children.
//
// See Also:
// 	Budget.RecursiveBalance
func (b Budget) Balance() int64 {
	return b.balance
}

// RecursiveBalance finds the balance of a `Budget` and all of its children.
//
// See Also:
// 	Budget.Balance
func (b Budget) RecursiveBalance() (sum int64) {
	sum = b.Balance()
	for _, child := range b.Children() {
		sum += child.RecursiveBalance()
	}
	return
}

// WithBalance creates a copy of a Budget but with the updated Balance.
//
// See Also:
// 	Budget.IncreaseBalance
// 	Budget.DecreaseBalance
func (b Budget) WithBalance(val int64) (updated Budget) {
	updated = b.deepCopy()
	updated.balance = val
	return
}

// IncreaseBalance creates a copy of a Budget but with the Balance credited by the
// specified amount.
//
// See Also:
// 	Budget.WithBalance
// 	Budget.DecreaseBalance
func (b Budget) IncreaseBalance(credit int64) Budget {
	return b.WithBalance(b.Balance() + credit)
}

// DecreaseBalance creates a copy of a Budget but with the Balance debited by the
// specified amount.
//
// See Also:
// 	Budget.WithBalance
// 	Budget.IncreaseBalance
func (b Budget) DecreaseBalance(debit int64) Budget {
	return b.WithBalance(b.Balance() - debit)
}

// ApplyEffect creates a Budget that is identical to the current one, but with
// the specified adjustments to the balance of each Budget.
func (b Budget) ApplyEffect(e Effect) (result Budget, err error) {
	panic("code not written")
}

// AddChild creates a copy of a Budget but with one addiotional child, should
// there not already have been a child with that name.
//
// See Also:
// 	Budget.RemoveChild
// 	Budget.SetChildren
func (b Budget) AddChild(name string, child Budget) (updated Budget, added bool) {
	if _, ok := b.children[name]; ok {
		updated = b
		return
	}

	updated = b.deepCopy()
	if updated.children == nil {
		updated.children = make(map[string]Budget, 1)
	}
	updated.children[name] = child
	added = true
	return
}

// RemoveChild creates a copy of a Budget, but with one fewer children, should
// it have had a child with the specified name.
//
// See Also:
// 	Budget.AddChild
// 	Budget.SetChildren
func (b Budget) RemoveChild(name string) (updated Budget, removed bool) {
	if _, ok := b.children[name]; !ok {
		updated = b
		return
	}

	updated = b.deepCopy()
	delete(updated.children, name)
	if len(updated.children) <= 0 {
		updated.children = nil
	}
	removed = true
	return
}

// Child retrieves the Budget that is a child of this Budget with the specified
// name.
func (b Budget) Child(name string) (child Budget, present bool) {
	child, present = b.children[name]
	return
}

// Children retrieves all child instances of Budget for a given Budget.
func (b Budget) Children() (results map[string]Budget) {
	results = make(map[string]Budget, len(b.children))

	for name, child := range b.children {
		results[name] = child
	}
	return
}

// WithChildren creates a copy of a Budget with an updated list of children.
//
// See Also:
// 	Budget.AddChild
// 	Budget.RemoveChild
func (b Budget) WithChildren(children map[string]Budget) (updated Budget) {
	updated.balance = b.balance
	updated.children = make(map[string]Budget, len(children))
	for name, child := range children {
		updated.children[name] = child
	}
	return updated
}

// MarshalJSON converts an in memory Budget to a JSON string.
func (b Budget) MarshalJSON() ([]byte, error) {
	builder := new(bytes.Buffer)

	var helper func(Budget)

	helper = func(b Budget) {
		builder.WriteRune('{')
		fmt.Fprintf(builder, "\"balance\":%d", b.Balance())
		if len(b.children) > 0 {
			builder.WriteString(`,"children":{`)
			children := b.Children()

			orderedChildren := make([]string, 0, len(children))
			for name := range children {
				orderedChildren = append(orderedChildren, name)
			}
			sort.Slice(orderedChildren, func(i, j int) bool { return orderedChildren[i] < orderedChildren[j] })

			for _, name := range orderedChildren {
				fmt.Fprintf(builder, "\"%s\":", name)
				helper(children[name])
				builder.WriteRune(',')
			}
			builder.Truncate(builder.Len() - 1)
			builder.WriteRune('}')
		}
		builder.WriteRune('}')
	}

	helper(b)
	return builder.Bytes(), nil
}

// UnmarshalJSON converts a series of bytes into an in memory Budget.
func (b *Budget) UnmarshalJSON(content []byte) (err error) {
	var intermediate map[string]json.RawMessage

	err = json.Unmarshal(content, &intermediate)
	if err != nil {
		return
	}

	b.balance, err = strconv.ParseInt(string(intermediate["balance"]), 10, 64)
	if err != nil {
		return
	}

	if _, ok := intermediate["children"]; ok {
		children := make(map[string]json.RawMessage)
		b.children = make(map[string]Budget)
		err = json.Unmarshal(intermediate["children"], &children)
		if err != nil {
			return
		}

		for name, childText := range children {
			var current Budget
			err = json.Unmarshal([]byte(childText), &current)
			if err != nil {
				return
			}

			b.children[name] = current
		}
	}
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
		currentBalance := float64(b.Balance()) / 100
		builder.WriteRune('{')

		if currentName != "" {
			builder.WriteString(currentName)
			builder.WriteRune(':')
		}
		fmt.Fprintf(builder, "$%0.2f", currentBalance)

		if len(b.Children()) > 0 {
			const childSuffix = ", "
			builder.WriteString(" [")
			for childName, child := range b.Children() {
				helper(childName, child)
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
