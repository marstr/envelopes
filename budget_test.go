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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/marstr/envelopes"
)

func ExampleBudget_SetBalance() {
	subject := envelopes.Budget{}
	updated := subject.WithBalance(1729)

	fmt.Println(subject.Balance())
	fmt.Println(updated.Balance())
	// Output:
	// 0
	// 1729
}

func ExampleBudget_IncreaseBalance() {
	subject := envelopes.Budget{}.WithBalance(100)
	subject = subject.IncreaseBalance(45)
	fmt.Println(subject.Balance())
	// Output: 145
}

func ExampleBudget_DecreaseBalance() {
	subject := envelopes.Budget{}.WithBalance(100)
	subject = subject.DecreaseBalance(99)
	fmt.Println(subject.Balance())
	// Output: 1
}

func ExampleBudget_AddChild() {
	subject := envelopes.Budget{}.WithBalance(42)
	updated, added := subject.AddChild("child1", envelopes.Budget{}.WithBalance(2))
	fmt.Println(subject)
	fmt.Println(updated)
	fmt.Println(added)
	// Output:
	// {$0.42}
	// {$0.42 [{child1:$0.02}]}
	// true
}

func ExampleBudget_RecursiveBalance() {
	subject := envelopes.Budget{}.WithBalance(431)
	subject, _ = subject.AddChild("child1", envelopes.Budget{}.WithBalance(1296))
	subject, _ = subject.AddChild("child2", envelopes.Budget{}.WithBalance(2))

	fmt.Println(subject.RecursiveBalance())
	// Output: 1729
}

func ExampleBudget_MarshalJSON() {
	subject := envelopes.Budget{}.WithBalance(42)
	subject, _ = subject.AddChild("child1", envelopes.Budget{}.WithBalance(9087))

	output, _ := json.Marshal(subject)
	fmt.Println(string(output))
	// Output:
	// {"balance":42,"children":{"child1":{"balance":9087}}}
}

func TestBudget_Equal(t *testing.T) {
	testCases := []struct {
		a        envelopes.Budget
		b        envelopes.Budget
		expected bool
	}{
		{
			envelopes.Budget{},
			envelopes.Budget{},
			true,
		},
		{
			envelopes.Budget{}.WithBalance(45),
			envelopes.Budget{}.WithBalance(45),
			true,
		},
		{
			envelopes.Budget{}.WithBalance(39),
			envelopes.Budget{}.WithBalance(90),
			false,
		},
		{
			envelopes.Budget{}.WithBalance(99),
			envelopes.Budget{}.WithBalance(0).WithChildren(map[string]envelopes.Budget{"child1": envelopes.Budget{}.WithBalance(99)}),
			false,
		},
		{
			envelopes.Budget{}.WithBalance(0).WithChildren(map[string]envelopes.Budget{"child1": envelopes.Budget{}.WithBalance(99)}),
			envelopes.Budget{}.WithBalance(0).WithChildren(map[string]envelopes.Budget{"child2": envelopes.Budget{}.WithBalance(99)}),
			false,
		},
		{
			envelopes.Budget{}.WithBalance(0).WithChildren(map[string]envelopes.Budget{"child1": envelopes.Budget{}.WithBalance(99)}),
			envelopes.Budget{}.WithBalance(0).WithChildren(map[string]envelopes.Budget{"child1": envelopes.Budget{}.WithBalance(44)}),
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s %s", tc.a, tc.b), func(t *testing.T) {
			if got := tc.a.Equal(tc.b); got != tc.expected {
				t.Fail()
			}

			if got := tc.b.Equal(tc.a); got != tc.expected {
				t.Fail()
			}
		})
	}
}

func TestBudget_UnmarshalJSON(t *testing.T) {
	testCases := []struct {
		string
		expected envelopes.Budget
	}{
		{
			`{"balance":0}`,
			envelopes.Budget{},
		},
		{
			`{"balance":1729}`,
			envelopes.Budget{}.WithBalance(1729),
		},
		{
			`{"balance":42,"children":{"child1":{"balance":2}}}`,
			envelopes.Budget{}.WithBalance(42).WithChildren(map[string]envelopes.Budget{
				"child1": envelopes.Budget{}.WithBalance(2),
			}),
		},
		{
			`{"balance":42,"children":{"child1":{"balance":2},"child2":{"balance":4}}}`,
			envelopes.Budget{}.WithBalance(42).WithChildren(map[string]envelopes.Budget{
				"child1": envelopes.Budget{}.WithBalance(2),
				"child2": envelopes.Budget{}.WithBalance(4),
			}),
		},
		{
			`{"balance":42,"children":{"child2":{"balance":4},"child1":{"balance":2}}}`,
			envelopes.Budget{}.WithBalance(42).WithChildren(map[string]envelopes.Budget{
				"child1": envelopes.Budget{}.WithBalance(2),
				"child2": envelopes.Budget{}.WithBalance(4),
			}),
		},
		{
			`{"children":{"child2":{"balance":4},"child1":{"balance":2}},"balance":42}`,
			envelopes.Budget{}.WithBalance(42).WithChildren(map[string]envelopes.Budget{
				"child1": envelopes.Budget{}.WithBalance(2),
				"child2": envelopes.Budget{}.WithBalance(4),
			}),
		},
		{
			`{"balance":42,"children":{"child1":{"balance":2},"child2":{"balance":4,"children":{"child3":{"balance":99}}}}}`,
			envelopes.Budget{}.WithBalance(42).WithChildren(map[string]envelopes.Budget{
				"child1": envelopes.Budget{}.WithBalance(2),
				"child2": envelopes.Budget{}.WithBalance(4).WithChildren(map[string]envelopes.Budget{
					"child3": envelopes.Budget{}.WithBalance(99),
				}),
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.string, func(t *testing.T) {
			var got envelopes.Budget
			err := json.Unmarshal([]byte(tc.string), &got)
			if err != nil {
				t.Error(err)
			}

			if !tc.expected.Equal(got) {
				t.Logf("\ngot:  %q\nwant: %q", got, tc.expected)
				t.Fail()
			}
		})
	}
}

func TestBudget_MarshalJSON(t *testing.T) {
	oneChild, _ := envelopes.Budget{}.WithBalance(42).AddChild("beta", envelopes.Budget{}.WithBalance(99))
	twoChildren, _ := oneChild.AddChild("alpha", envelopes.Budget{}.WithBalance(56))

	testCases := []struct {
		envelopes.Budget
		want string
	}{
		{
			envelopes.Budget{},
			`{"balance":0}`,
		},
		{
			oneChild,
			`{"balance":42,"children":{"beta":{"balance":99}}}`,
		},
		{
			twoChildren,
			`{"balance":42,"children":{"alpha":{"balance":56},"beta":{"balance":99}}}`,
		},
		{
			envelopes.Budget{}.WithBalance(-98).WithChildren(map[string]envelopes.Budget{
				"child1": envelopes.Budget{}.WithBalance(87),
				"child2": envelopes.Budget{}.WithBalance(11),
			}),
			`{"balance":-98,"children":{"child1":{"balance":87},"child2":{"balance":11}}}`,
		},
	}

	for _, tc := range testCases {
		got, err := json.Marshal(tc.Budget)
		if err != nil {
			t.Error(err)
		}

		if string(got) != tc.want {
			t.Logf("\ngot:  \t%s\nwant: \t%s", got, tc.want)
			t.Fail()
		}
	}
}

func TestBudget_ID_Deterministic(t *testing.T) {
	testCases := []envelopes.Budget{
		envelopes.Budget{},
		envelopes.Budget{}.WithBalance(1729),
		envelopes.Budget{}.WithChildren(map[string]envelopes.Budget{
			"child1":     envelopes.Budget{}.WithBalance(99),
			"alphaChild": envelopes.Budget{}.WithBalance(44),
		}),
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

func TestBudget_JSONRoundTrip(t *testing.T) {
	testCases := []envelopes.Budget{
		envelopes.Budget{},
		envelopes.Budget{}.WithBalance(42),
		envelopes.Budget{}.WithBalance(1729).WithChildren(map[string]envelopes.Budget{
			"child1": envelopes.Budget{}.WithBalance(99),
			"child2": envelopes.Budget{}.WithBalance(1),
		}),
		envelopes.Budget{}.WithBalance(1729).WithChildren(map[string]envelopes.Budget{
			"child1": envelopes.Budget{}.WithBalance(99).WithChildren(map[string]envelopes.Budget{
				"child2": envelopes.Budget{}.WithBalance(1),
			}),
		}),
		envelopes.Budget{}.WithBalance(1729).WithChildren(map[string]envelopes.Budget{
			"child1": envelopes.Budget{}.WithBalance(-1007).WithChildren(map[string]envelopes.Budget{
				"child2": envelopes.Budget{}.WithBalance(1),
			}),
		}),
	}

	for _, tc := range testCases {
		t.Run(tc.String(), func(t *testing.T) {
			marshaled, err := json.Marshal(tc)
			if err != nil {
				t.Error(err)
			}

			var unmarshaled envelopes.Budget
			err = json.Unmarshal(marshaled, &unmarshaled)
			if err != nil {
				t.Error(err)
			}

			if !tc.Equal(unmarshaled) {
				t.Fail()
			}
		})
	}
}
