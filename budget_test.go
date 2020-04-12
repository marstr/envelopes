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

package envelopes_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/marstr/envelopes"
)

func ExampleBudget_RecursiveBalance() {
	subject := envelopes.Budget{
		Balance: envelopes.Balance{"USD": big.NewRat(431, 100)},
		Children: map[string]*envelopes.Budget{
			"child1": {Balance: envelopes.Balance{"USD": big.NewRat(1296, 100)}},
			"child2": {Balance: envelopes.Balance{"USD": big.NewRat(2, 100)}},
		},
	}

	fmt.Println(subject.RecursiveBalance())
	// Output: USD 17.290
}

func TestBudget_Equal(t *testing.T) {
	testCases := []struct {
		a        envelopes.Budget
		b        envelopes.Budget
		expected bool
	}{
		{
			envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(45, 100)}},
			envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(45, 100)}},
			true,
		},
		{
			envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(39, 100)}},
			envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(90, 100)}},
			false,
		},
		{
			envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(99, 100)}},
			envelopes.Budget{Children: map[string]*envelopes.Budget{"child1": {Balance: envelopes.Balance{"USD": big.NewRat(99, 100)}}}},
			false,
		},
		{
			envelopes.Budget{Children: map[string]*envelopes.Budget{"child1": {Balance: envelopes.Balance{"USD": big.NewRat(99, 100)}}}},
			envelopes.Budget{Children: map[string]*envelopes.Budget{"child2": {Balance: envelopes.Balance{"USD": big.NewRat(99, 100)}}}},
			false,
		},
		{
			envelopes.Budget{Children: map[string]*envelopes.Budget{"child1": {Balance: envelopes.Balance{"USD": big.NewRat(99, 100)}}}},
			envelopes.Budget{Children: map[string]*envelopes.Budget{"child1": {Balance: envelopes.Balance{"USD": big.NewRat(44, 100)}}}},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			if got := tc.a.Equal(tc.b); got != tc.expected {
				t.Fail()
			}

			if got := tc.b.Equal(tc.a); got != tc.expected {
				t.Fail()
			}
		})
	}
}

func TestBudget_ID_Deterministic(t *testing.T) {
	testCases := []*envelopes.Budget{
		{},
		{Balance: envelopes.Balance{"USD": big.NewRat(1729, 100)}},
		{
			Children: map[string]*envelopes.Budget{
				"child1":     {Balance: envelopes.Balance{"USD": big.NewRat(99, 100)}},
				"alphaChild": {Balance: envelopes.Balance{"USD": big.NewRat(44, 100)}},
			},
		},
	}

	for _, tc := range testCases {
		initial := tc.ID()
		t.Run(fmt.Sprintf("%x", initial), func(t *testing.T) {
			for i := 0; i < 30; i++ {
				subsequent := tc.ID()

				for j, entry := range initial {
					if subsequent[j] != entry {
						t.Logf("Subsequent: %x", subsequent)
						t.FailNow()
					}
				}
			}
		})
	}
}

func TestBudget_ID_Lock(t *testing.T) {
	// This is a list of well-known budgets and their corresponding IDs. Should you change any of these test cases,
	// it is indicative that these changes have broken all existing repositories and that a migration will need to
	testCases := map[string]envelopes.Budget{
		"788245b186cad464b7aa1e8e359eb19fbcf7b6e4": {},
		"17c8c20300bdd02c076cc5cd3b67139a1e203713": {
			Balance: envelopes.Balance{"USD": big.NewRat(1098, 100)},
			Children: map[string]*envelopes.Budget{
				"grocery": {
					Balance: envelopes.Balance{"USD": big.NewRat(4598, 100)},
				},
				"restaurants": {
					Balance: envelopes.Balance{"USD": big.NewRat(9978, 100)},
				},
			},
		},
	}

	for expected, subject := range testCases {
		got := subject.ID()

		var want envelopes.ID

		err := want.UnmarshalText([]byte(expected))
		if err != nil {
			t.Errorf("could not unmarshal %q: %v", expected, err)
			continue
		}

		if !got.Equal(want) {
			t.Logf("\ngot: \t%s\nwant:\t%s", got, want)
			t.Fail()
		}
	}
}
