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

// "can't let the evil of the money trap me"
// -Holla at Me, Tupac Shakur

package envelopes

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"sort"
)

// Accounts collects all pools of money that can be spent or deposited to.
type Accounts map[string]Balance

// ID fetches a hash of this combinations of accounts with their balances.
func (accs Accounts) ID() ID {
	accountNames := accs.Names()

	// Fetch, clear, and promise to return a buffer to hold the ID defining characteristics of this IDer.
	identityBuilder := identityBuilders.Get().(*bytes.Buffer)
	identityBuilder.Reset()
	defer identityBuilders.Put(identityBuilder)

	// Create a raw list of each component relevant to building an ID.
	for i := range accountNames {
		fmt.Fprintf(identityBuilder, "account %s %s\n", accountNames[i], accs[accountNames[i]])
	}

	// Aggregate and set the ID of this IDer
	return sha1.Sum(identityBuilder.Bytes())
}

// Names fetches the distinct account names represented in this structure.
func (accs Accounts) Names() (names []string) {
	names = make([]string, 0, len(accs))

	for name := range accs {
		names = append(names, name)
	}

	sort.Strings(names)

	return
}

// RenameAccount changes the name associated with an account. If the new name already exists in this collection of
// accounts, nothing happens and this function return the original object and false.
func (accs Accounts) RenameAccount(old, new string) bool {
	if accs.HasAccount(new) || !accs.HasAccount(old) {
		return false
	}

	accs[new] = accs[old]
	delete(accs, old)
	return true
}

// HasAccount determines whether or not an account exists with the desired name.s
func (accs Accounts) HasAccount(name string) (ok bool) {
	_, ok = accs[name]
	return
}

// Balance finds the total balance of all accounts in this collection.
func (accs Accounts) Balance() Balance {
	sum := make(Balance)
	for _, bal := range accs {
		sum = sum.Add(bal)
	}
	return sum
}

// Add combines two lists of accounts. Taking the union of all distinct accounts, and where the same account appears in two places, it sums both balances.
func (accs Accounts) Add(other Accounts) Accounts {
	modifiedAccounts := make(Accounts, len(accs))

	unseen := make(Accounts, len(other))
	for otherName, otherBalance := range other {
		unseen[otherName] = otherBalance
	}

	for currentName, currentBalance := range accs {
		if otherBalance, ok := other[currentName]; ok {
			delete(unseen, currentName)
			if !currentBalance.Equal(otherBalance) {
				modifiedAccounts[currentName] = currentBalance.Add(otherBalance)
			}
		} else {
			modifiedAccounts[currentName] = currentBalance
		}
	}

	for unseenName, unseenBalance := range unseen {
		modifiedAccounts[unseenName] = unseenBalance
	}

	return modifiedAccounts
}

// Sub removes the amount from each `Balance` in other from the account it is invoked on.
func (accs Accounts) Sub(other Accounts) Accounts {
	modifiedAccounts := make(Accounts, len(accs))

	unseen := make(Accounts, len(other))
	for otherName, otherBalance := range other {
		unseen[otherName] = otherBalance
	}

	for currentName, currentBalance := range accs {
		if otherBalance, ok := other[currentName]; ok {
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
