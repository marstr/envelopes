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
	"testing"

	"github.com/marstr/envelopes"
)

func ExampleEffect_Add() {
	var a envelopes.Budget
	a.Name = "A"

	left := envelopes.Effect{
		&a: 2,
	}

	right := envelopes.Effect{
		&a: 4,
	}

	updated := left.Add(right)

	fmt.Println("Left", left)
	fmt.Println("Right", right)
	fmt.Println("Updated", updated)
	// Output:
	// Left ["A":$0.02]
	// Right ["A":$0.04]
	// Updated ["A":$0.06]
}

func ExampleEffect_Truncate() {
	eff := envelopes.Effect{
		&envelopes.Budget{Name: "Groceries"}:     9907,
		&envelopes.Budget{Name: "Transit"}:       504,
		&envelopes.Budget{Name: "Miscellaneous"}: 0,
	}

	fmt.Println(eff)
	eff.Truncate()
	fmt.Println(eff)
	// Output:
	// ["Groceries":$99.07 "Transit":$5.04 "Miscellaneous":$0.00]
	// ["Groceries":$99.07 "Transit":$5.04]
}

func TestEffect_String(t *testing.T) {
	groceries := envelopes.Budget{
		Name:    "Groceries",
		Balance: 10399,
	}

	transit := envelopes.Budget{
		Name:    "Transit",
		Balance: 4297,
	}

	testCases := []struct {
		envelopes.Effect
		want string
	}{
		{make(envelopes.Effect), "[]"},
		{
			envelopes.Effect{
				&groceries: 10,
				&transit:   -3,
			},
			`["Groceries":$0.10 "Transit":$-0.03]`,
		},
		{
			envelopes.Effect{
				&groceries: 1000,
				&transit:   -1200,
			},
			`["Transit":$-12.00 "Groceries":$10.00]`,
		},
		{
			envelopes.Effect{
				nil: 4,
			},
			`[nil:$0.04]`,
		},
		{
			envelopes.Effect{
				nil:                          2271,
				&groceries:                   9910,
				&transit:                     2091,
				&envelopes.Budget{Name: "A"}: 1,
				&envelopes.Budget{Name: "B"}: 2,
				&envelopes.Budget{Name: "C"}: 3,
				&envelopes.Budget{Name: "D"}: 4,
				&envelopes.Budget{Name: "E"}: 5,
				&envelopes.Budget{Name: "F"}: 6,
				&envelopes.Budget{Name: "G"}: 7,
				&envelopes.Budget{Name: "H"}: 8,
				&envelopes.Budget{Name: "I"}: 9,
				&envelopes.Budget{Name: "J"}: 10,
				&envelopes.Budget{Name: "K"}: 11,
				&envelopes.Budget{Name: "L"}: 12,
				&envelopes.Budget{Name: "M"}: 13,
				&envelopes.Budget{Name: "N"}: 14,
			},
			`["Groceries":$99.10 nil:$22.71 "Transit":$20.91 "N":$0.14 "M":$0.13 "L":$0.12 "K":$0.11 "J":$0.10 "I":$0.09 "H":$0.08 "G":$0.07 "F":$0.06 "E":$0.05 "D":$0.04 "C":$0.03 ...]`,
		},
	}

	for _, tc := range testCases {
		if got := tc.String(); got != tc.want {
			t.Logf("\ngot:  %q\nwant: %q", got, tc.want)
			t.Fail()
		}
	}
}
