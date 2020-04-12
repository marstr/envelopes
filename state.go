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

package envelopes

import (
	"bytes"
	"crypto/sha1"
	"fmt"
)

type (
	// State captures the values of all Budgets and Accounts.
	State struct {
		Budget   *Budget
		Accounts Accounts
	}

	// Impact captures the difference between two States.
	Impact State
)

// ID calculates the SHA1 hash of this object.
func (s State) ID() (id ID) {
	marshaled, err := s.MarshalText()
	if err != nil {
		return ID{}
	}
	return sha1.Sum(marshaled)
}

// Equal determines whether or not each component of two States have the same balances. If any components are not
// shared, the answer is false.
func (s State) Equal(other State) bool {
	result := s.Subtract(other)
	if len(result.Accounts) > 0 {
		return false
	}

	if result.Budget == nil {
		return true
	}

	return result.Budget.Equal(Budget{})
}

// MarshalText computes a deterministic string that uniquely represents this State.
func (s State) MarshalText() ([]byte, error) {

	identityBuilder := identityBuilders.Get().(*bytes.Buffer)
	identityBuilder.Reset()
	defer identityBuilders.Put(identityBuilder)

	if s.Budget == nil {
		s.Budget = &Budget{}
	}
	_, err := fmt.Fprintf(identityBuilder, "budget %s\n", s.Budget.ID())
	if err != nil {
		return nil, err
	}

	if s.Accounts == nil {
		s.Accounts = make(Accounts, 0)
	}
	_, err = fmt.Fprintf(identityBuilder, "accounts %s\n", s.Accounts.ID())
	if err != nil {
		return nil, err
	}

	return identityBuilder.Bytes(), nil
}

func (s State) String() string {
	id := s.ID()
	return string(id[:])
}

// Subtract removes the balances of another State, and returns what changed between the two.
func (s State) Subtract(other State) Impact {
	return Impact{
		Accounts: s.subtractAccounts(other),
		Budget:   s.subtractBudget(other),
	}
}

func (s State) subtractAccounts(other State) Accounts {
	modifiedAccounts := make(Accounts, len(s.Accounts))

	unseen := make(Accounts, len(other.Accounts))
	for otherName, otherBalance := range other.Accounts {
		unseen[otherName] = otherBalance
	}

	for currentName, currentBalance := range s.Accounts {
		if otherBalance, ok := other.Accounts[currentName]; ok {
			delete(unseen, currentName)
			if !currentBalance.Equal(otherBalance) {
				modifiedAccounts[currentName] = currentBalance.Sub(otherBalance)
			}
		} else {
			modifiedAccounts[currentName] = currentBalance
		}
	}

	for unseenName, unseenBalance := range unseen {
		modifiedAccounts[unseenName] = unseenBalance.Negate()
	}

	return modifiedAccounts
}

func (s State) subtractBudget(other State) *Budget {
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
			childClone := child.deepCopy()
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
	if s.Budget == nil && other.Budget == nil {
		return nil
	}

	// If this has a budget, but the other doesn't, just clone this budget. This budget has been added.
	if other.Budget == nil {
		cloned := s.Budget.deepCopy()
		return &cloned
	}

	// If this doesn't have a budget, but the other does, clone and negate that budget. That budget has been removed.
	if s.Budget == nil {
		cloned := other.Budget.deepCopy()
		negate(&cloned)
		return &cloned
	}

	// Changes have been made to this budget. Call helper to get the differences figured out.
	return helper(s.Budget, other.Budget)
}
