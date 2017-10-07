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

func TestBudget_RecursiveTotal(t *testing.T) {
	testCases := []struct {
		*envelopes.Budget
		Expected int64
	}{
		{
			Budget:   &envelopes.Budget{Balance: 42},
			Expected: 42,
		},
		{
			Budget: &envelopes.Budget{
				Balance: 42,
				Children: []*envelopes.Budget{
					&envelopes.Budget{Balance: 89},
				},
			},
			Expected: 131,
		},
		{
			Budget: &envelopes.Budget{
				Balance: 42,
				Children: []*envelopes.Budget{
					&envelopes.Budget{Balance: 89},
					&envelopes.Budget{Balance: 1006},
				},
			},
			Expected: 1137,
		},
		{
			Budget: &envelopes.Budget{
				Balance: 42,
				Children: []*envelopes.Budget{
					&envelopes.Budget{
						Balance: 89,
						Children: []*envelopes.Budget{
							&envelopes.Budget{Balance: 99},
							&envelopes.Budget{Balance: 227},
						},
					},
					&envelopes.Budget{Balance: 1006},
				},
			},
			Expected: 1463,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.String(), func(t *testing.T) {
			result := tc.RecursiveBalance()
			if result != tc.Expected {
				t.Logf("got: %d want: %d", result, tc.Expected)
				t.Fail()
			}
		})
	}
}

func ExampleBudget_RecursiveBalance() {
	subject := envelopes.Budget{
		Balance: 431,
		Children: []*envelopes.Budget{
			&envelopes.Budget{Balance: 1296},
			&envelopes.Budget{Balance: 2},
		},
	}

	fmt.Println(subject.RecursiveBalance())
	// Output: 1729
}

func TestBudget_String(t *testing.T) {
	testCases := []struct {
		specimen envelopes.Budget
		expected string
	}{
		{
			specimen: envelopes.Budget{Name: "Dirty Harry", Balance: 900123},
			expected: `{"Dirty Harry":$9001.23}`,
		},
		{
			specimen: envelopes.Budget{
				Name:    "Harry Potter",
				Balance: 42576,
				Children: []*envelopes.Budget{
					&envelopes.Budget{
						Name:    "Ron Weasley",
						Balance: 02,
					},
					&envelopes.Budget{
						Name:    "Hermione Granger",
						Balance: 523471,
					},
				},
			},
			expected: `{"Harry Potter":$425.76 [{"Ron Weasley":$0.02}, {"Hermione Granger":$5234.71}]}`,
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			result := tc.specimen.String()

			if result != tc.expected {
				t.Logf("\ngot: \t%q\nwant:\t%q", result, tc.expected)
				t.Fail()
			}
		})
	}
}
