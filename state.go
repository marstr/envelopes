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
	"fmt"
	"strings"
)

// State records the balances of a collection of budgets and accounts at a given moment in time.
type State struct {
	Budgets  []*Budget
	Accounts []*Account
	Parent   *State
}

// func (s *State) ApplyEffect(eff Effect) (err error) {
// 	for budg := range s.Budgets {

// 	}
// }

// func (s *State) ApplyTransaction(trans Transaction) (err error) {

// }

func (s State) FindBudget(specifier string) (found *Budget, err error) {
	var descend func(*Budget, []string) *Budget
	descend = func(current *Budget, remaining []string) *Budget {
		if len(remaining) == 0 {
			return current
		}

		for _, child := range current.Children {
			if child.Name == remaining[0] {
				return descend(child, remaining[1:])
			}
		}
		return nil
	}

	for _, child := range s.Budgets {
		if result := descend(child, strings.Split(specifier, "/")); result != nil {
			found = result
			return
		}
	}
	err = ErrorBudgetNotFound{
		Target:    s,
		Specifier: specifier,
	}
	return
}

type ErrorBudgetNotFound struct {
	Target    State
	Specifier string
}

func (bnf ErrorBudgetNotFound) Error() string {
	return fmt.Sprintf("Could not find specifier: %q", bnf.Specifier)
}
