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

// DeepCopy creates a new budget identical used to the one to invoke it, but
// that is a totally separate instance. This allows you to edit the copied or
// original without impacting the other.
func (b Budget) DeepCopy() Budget {
	var clone Budget
	clone.Balance = b.Balance
	clone.Children = make(map[string]*Budget, len(b.Children))

	for childName, child := range b.Children {
		clonedChild := child.DeepCopy()
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
		_, err = fmt.Fprintf(identityBuilder, "\nchild %s %s", childName, childID)
		if err != nil {
			return nil, err
		}
	}

	return identityBuilder.Bytes(), nil
}

// Equal determines whether or not two instances of Budget share the same balance and
// have children that are all equal and have the same name.
func (b Budget) Equal(other Budget) bool {
	if !b.Balance.Equal(other.Balance) {
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
//
//	Budget.Balance
func (b Budget) RecursiveBalance() (sum Balance) {
	sum = b.Balance
	for _, child := range b.Children {
		sum = sum.Add(child.RecursiveBalance())
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

// Subtract removes the amount of each balance in `other` from the Budget this was invoked on.
func (b *Budget) Subtract(other *Budget) *Budget {
	// negate reverses the balances of all balances in a budget recursively.
	var negate func(*Budget)
	negate = func(subject *Budget) {
		subject.Balance = subject.Balance.Negate()
		for _, child := range subject.Children {
			negate(child)
		}
	}

	// helper subtracts the right child-budget from the left child-budget. Should be called on children with matching
	// names
	var helper func(*Budget, *Budget) *Budget
	helper = func(left, right *Budget) *Budget {
		modifiedChildren := make(map[string]*Budget)

		// mark all of the children on the right, so we can decide if any got deleted after tallying all of the children
		// on the left.
		removedChildren := make(map[string]*Budget, len(right.Children))
		for childName, child := range right.Children {
			removedChildren[childName] = child
		}

		// enumerate through all of the left's children, to find any discrepancies.
		for childName, leftChild := range left.Children {
			if rightChild, ok := right.Children[childName]; ok {
				// see if there are any differences between the children, only if there are should the modified child
				// be added.
				subtracted := helper(leftChild, rightChild)
				if subtracted != nil {
					modifiedChildren[childName] = subtracted
				}

				// regardless of whether or not there are differences, both budgets had this child.
				delete(removedChildren, childName)
			} else {
				modifiedChildren[childName] = leftChild
			}
		}

		// make a note of all of the children who were removed.
		for childName, child := range removedChildren {
			childClone := child.DeepCopy()
			negate(&childClone)
			modifiedChildren[childName] = &childClone
		}

		// finalize an object that represents the changes made, and send it up the stack.
		var retval Budget

		if len(modifiedChildren) > 0 {
			retval.Children = modifiedChildren
		}

		if left.Balance.Equal(right.Balance) && retval.Children == nil {
			return nil
		}

		retval.Balance = left.Balance.Sub(right.Balance)

		return &retval
	}

	// If there are no budgets involved... do nothing and short-circuit.
	if b == nil && other == nil {
		return nil
	}

	// If this has a budget, but the other doesn't, just clone this budget. This budget has been added.
	if other == nil {
		cloned := b.DeepCopy()
		return &cloned
	}

	// If this doesn't have a budget, but the other does, clone and negate that budget. That budget has been removed.
	if b == nil {
		cloned := other.DeepCopy()
		negate(&cloned)
		return &cloned
	}

	// Changes have been made to this budget. Call helper to get the differences figured out.
	return helper(b, other)
}
