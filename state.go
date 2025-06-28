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
	"math/big"
	"os"
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

// Equal determines whether or not each component of two Impacts have the same balances. If any components are not
// shared, the answer is false.
func (i Impact) Equal(other Impact) bool {
	return State(i).Equal(State(other))
}

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
		Accounts: s.Accounts.Sub(other.Accounts),
		Budget:   s.Budget.Subtract(other.Budget),
	}
}

// Add combines the baalances of another State and returns the total of both.
func (s State) Add(other State) Impact {
	return Impact{
		Accounts: s.Accounts.Add(other.Accounts),
		Budget:   s.Budget.Add(other.Budget),
	}
}

// CalculateAmount looks at the difference between two states, and boils down the changes
// into a single number that captures the magnitude of the operation(s) that occured between
// the two.
//
// For simple cases, that number is quite obvious. If you buy a coffee for three
// bucks, the Amount suggested here should be three bucks. But other examples are trickier.
// Shifting money from one part of your budget to another? Is that zero since no money moved
// accounts, or is it the amount that moved? The answer doesn't necessarily matter since the
// full truth can be found by looking at the difference between the two states. But having a
// single consistent answer that is recommended by this platform has value and will help
// shape users intuition over time. In the earlier example, the library would argue that the
// more useful answer is the amount that moved between budgets, so thats what this function
// returns.
func CalculateAmount(original, updated State) Balance {
	if changed := findAccountAmount(original, updated); !changed.Equal(Balance{}) {
		return changed
	}

	return findBudgetAmount(original, updated)
}

func findAccountAmount(original, updated State) Balance {
	zero := big.NewRat(0, 1)

	modifiedAccounts := make(Accounts, len(original.Accounts))

	addedAccountNames := make(map[string]struct{}, len(original.Accounts))
	for name := range updated.Accounts {
		addedAccountNames[name] = struct{}{}
	}

	for name, oldBalance := range original.Accounts {
		delete(addedAccountNames, name)

		if newBalance, ok := updated.Accounts[name]; ok && newBalance.Equal(oldBalance) {
			// Nothing has changed
			continue
		} else if !ok {
			// An account was removed
			modifiedAccounts[name] = oldBalance.Negate()
		} else {
			// An account had its balance modified
			modifiedAccounts[name] = newBalance.Sub(oldBalance)
		}
	}

	// Iterate over the accounts that weren't seen in the original, and mark them as new.
	for name := range addedAccountNames {
		modifiedAccounts[name] = updated.Accounts[name]
	}

	// If there was a transfer between two accounts, we don't want to mark it as amount USD 0.00, but rather that magnitude
	// of the transfer. For that reason, we'll figure out the total negative and positive change of the accounts
	// involved.
	//
	// If it was a transfer between budgets, we'll count the total deposited into the receiving accounts.
	// If it was a deposit or credit, the amount positive or negative will get reflected because the opposite will
	// register as a zero.
	positiveAccountDifferences := make(Balance)
	negativeAccountDifferences := make(Balance)
	for _, bal := range modifiedAccounts {
		for asset, magnitude := range bal {
			if magnitude.Cmp(zero) > 0 {
				incrementAsset(positiveAccountDifferences, asset, magnitude)
			} else {
				incrementAsset(negativeAccountDifferences, asset, magnitude)
			}
		}
	}

	return combineSeparated(negativeAccountDifferences, positiveAccountDifferences)
}

func findBudgetAmount(original, updated State) Balance {
	zero := big.NewRat(0, 1)
	// Normalize the budgets into a flattened shape for easier comparison, more like Accounts
	const separator = string(os.PathSeparator)
	originalBudgets := make(map[string]Balance)
	updatedBudgets := make(map[string]Balance)

	var treeFlattener func(map[string]Balance, string, *Budget)
	treeFlattener = func(discovered map[string]Balance, currentPath string, target *Budget) {
		discovered[currentPath] = target.Balance

		for name, subTarget := range target.Children {
			treeFlattener(discovered, currentPath+separator+name, subTarget)
		}
	}

	treeFlattener(originalBudgets, separator, original.Budget)
	treeFlattener(updatedBudgets, separator, updated.Budget)

	// Make a list of all budget names in the updated state, so that we can find the ones which were added.
	addedBudgets := make(map[string]struct{}, len(updatedBudgets))
	for name := range updatedBudgets {
		addedBudgets[name] = struct{}{}
	}

	modifiedBudgets := make(map[string]Balance, len(originalBudgets))

	for name, oldBalance := range originalBudgets {
		delete(addedBudgets, name)

		if newBalance, ok := updatedBudgets[name]; ok && newBalance.Equal(oldBalance) {
			// Nothing has changed here
			continue
		} else if !ok {
			modifiedBudgets[name] = oldBalance.Negate()
		} else {
			modifiedBudgets[name] = newBalance.Sub(oldBalance)
		}
	}

	for name := range addedBudgets {
		modifiedBudgets[name] = updatedBudgets[name]
	}

	positiveBudgetDifferences := make(Balance)
	negativeBudgetDifferences := make(Balance)
	for _, bal := range modifiedBudgets {
		for asset, magnitude := range bal {
			if magnitude.Cmp(zero) > 0 {
				incrementAsset(positiveBudgetDifferences, asset, magnitude)
			} else {
				incrementAsset(negativeBudgetDifferences, asset, magnitude)
			}
		}
	}

	return combineSeparated(negativeBudgetDifferences, positiveBudgetDifferences)
}

func combineSeparated(negative, positive Balance) Balance {
	result := Balance{}

	for k, v := range negative {
		result[k] = v
	}

	for k, v := range positive {
		result[k] = v
	}

	return result
}

func incrementAsset(target Balance, asset AssetType, magnitude *big.Rat) {
	if prev, ok := target[asset]; ok {
		prev.Add(prev, magnitude)
	} else {
		target[asset] = magnitude
	}
}
