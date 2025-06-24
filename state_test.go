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
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/marstr/envelopes"
)

func TestState_ID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	t.Run("deterministic", getTestStateIDDeterministic(ctx))
	t.Run("lock", getTestStateIDLock(ctx))
}

func getTestStateIDDeterministic(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		testCases := []envelopes.State{
			{},
			{Budget: &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(1729, 100)}}},
		}

		for _, tc := range testCases {
			first := tc.ID()
			for i := 0; i < 30; i++ {
				select {
				case <-ctx.Done():
					t.Error(ctx.Err())
					return
				default:
					// Intentionally Left Blank
				}

				subsequent := tc.ID()

				if !subsequent.Equal(first) {
					t.Logf("subsuquent ID (%s) did not match initial ID (%s)", subsequent, first)
					t.Fail()
					continue
				}
			}
		}
	}
}

func getTestStateIDLock(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		testCases := []struct {
			Subject  envelopes.State
			Expected string
		}{
			{
				Subject:  envelopes.State{},
				Expected: "94ed54bf9f02dff56f806a2b6b853124e3608596",
			},
			{
				Subject: envelopes.State{
					Budget: &envelopes.Budget{
						Balance: envelopes.Balance{"USD": big.NewRat(4507, 100)},
						Children: map[string]*envelopes.Budget{
							"grocery":     {Balance: envelopes.Balance{"USD": big.NewRat(9813, 100)}},
							"restaurants": {Balance: envelopes.Balance{"USD": big.NewRat(10099, 100)}},
						},
					},
				},
				Expected: "18bfde2be25467512ad843ee8fccc707b5f036d7",
			},
		}

		for _, tc := range testCases {
			got := tc.Subject.ID().String()

			if got != tc.Expected {
				t.Logf("\ngot:  %q\nwant: %q", got, tc.Expected)
				t.Fail()
			}
		}
	}
}

func TestState_Subtract(t *testing.T) {
	testCases := []struct {
		Name     string
		Subject  envelopes.State
		Other    envelopes.State
		Expected envelopes.Impact
	}{
		{
			Name: "accounts_only",
			Subject: envelopes.State{
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(150, 1)},
				},
			},
			Other: envelopes.State{
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(100, 1)},
				},
			},
			Expected: envelopes.Impact{
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(50, 1)},
				},
			},
		},
		{
			Name: "unimpacted_accounts",
			Subject: envelopes.State{
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(150, 1)},
					"savings":  envelopes.Balance{"USD": big.NewRat(10000, 1)},
				},
			},
			Other: envelopes.State{
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(100, 1)},
					"savings":  envelopes.Balance{"USD": big.NewRat(10000, 1)},
				},
			},
			Expected: envelopes.Impact{
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(50, 1)},
				},
			},
		},
		{
			Name: "budget_only_simple_balance",
			Subject: envelopes.State{
				Budget: &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(5, 100)}},
			},
			Other: envelopes.State{
				Budget: &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(2, 100)}},
			},
			Expected: envelopes.Impact{
				Budget: &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(3, 100)}},
			},
		},
		{
			Name: "budget_only_recursive_difference",
			Subject: envelopes.State{
				Budget: &envelopes.Budget{
					Balance: envelopes.Balance{"USD": big.NewRat(9999, 100)},
					Children: map[string]*envelopes.Budget{
						"entertainment": {Balance: envelopes.Balance{"USD": big.NewRat(22, 1)}},
						"food": {
							Children: map[string]*envelopes.Budget{
								"restaurants": {Balance: envelopes.Balance{"USD": big.NewRat(19003, 100)}},
								"groceries":   {Balance: envelopes.Balance{"USD": big.NewRat(5307, 100)}},
							},
						},
					},
				},
			},
			Other: envelopes.State{
				Budget: &envelopes.Budget{
					Balance: envelopes.Balance{"USD": big.NewRat(9999, 100)},
					Children: map[string]*envelopes.Budget{
						"entertainment": {Balance: envelopes.Balance{"USD": big.NewRat(22, 1)}},
						"food": {
							Children: map[string]*envelopes.Budget{
								"restaurants": {Balance: envelopes.Balance{"USD": big.NewRat(19003, 100)}},
								"groceries":   {Balance: envelopes.Balance{"USD": big.NewRat(7807, 100)}},
							},
						},
					},
				},
			},
			Expected: envelopes.Impact{
				Budget: &envelopes.Budget{
					Children: map[string]*envelopes.Budget{
						"food": {
							Children: map[string]*envelopes.Budget{
								"groceries": {Balance: envelopes.Balance{"USD": big.NewRat(-25, 1)}},
							},
						},
					},
				},
			},
		},
		{
			Name: "budget_only_child_renamed",
			Subject: envelopes.State{
				Budget: &envelopes.Budget{
					Children: map[string]*envelopes.Budget{
						"foo": {Balance: envelopes.Balance{"USD": big.NewRat(50, 100)}},
					},
				},
			},
			Other: envelopes.State{
				Budget: &envelopes.Budget{
					Children: map[string]*envelopes.Budget{
						"bar": {Balance: envelopes.Balance{"USD": big.NewRat(50, 100)}},
					},
				},
			},
			Expected: envelopes.Impact{
				Budget: &envelopes.Budget{
					Children: map[string]*envelopes.Budget{
						"foo": {Balance: envelopes.Balance{"USD": big.NewRat(50, 100)}},
						"bar": {Balance: envelopes.Balance{"USD": big.NewRat(-50, 100)}},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		result := tc.Subject.Subtract(tc.Other)

		want := envelopes.State(tc.Expected)
		got := envelopes.State(result)

		if !got.Equal(want) {
			t.Fail()
			gotRawMarshaled, err := got.MarshalText()
			if err != nil {
				t.Error(err)
				continue
			}

			wantRawMarshaled, err := want.MarshalText()
			if err != nil {
				t.Error(err)
				continue
			}

			t.Logf("\ntest case: %q\ngot:\n%s\nwant:\n%s\n", tc.Name, string(gotRawMarshaled), string(wantRawMarshaled))
		}
	}
}

func TestCalculateAmount(t *testing.T) {
	t.Run("simpleDebit", testSimpleDebitsAndCredits(10, 7))
	t.Run("simpleCredit", testSimpleDebitsAndCredits(7, 10))
	t.Run("growPastZero", testSimpleDebitsAndCredits(-100, 100))
	t.Run("fallPastZero", testSimpleDebitsAndCredits(100, -100))
	t.Run("allNegative", testSimpleDebitsAndCredits(-100, -50))

	t.Run("renameBudget", renameBudget)
	t.Run("distributeFunds", distributeFunds)
	t.Run("threeWay", threeWay)
}

func testSimpleDebitsAndCredits(before, after int64) func(*testing.T) {
	return func(t *testing.T) {
		const currency = "USD"
		const accName = "checking"
		beforeBal := big.NewRat(before, 1)
		afterBal := big.NewRat(after, 1)

		expectedBal := big.NewRat(0, 1)
		expectedBal.Sub(afterBal, beforeBal)
		expected := envelopes.Balance{currency: expectedBal}

		preOp := envelopes.State{
			Accounts: envelopes.Accounts{
				accName: envelopes.Balance{currency: beforeBal},
			},
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{currency: beforeBal},
			},
		}

		postOp := envelopes.State{
			Accounts: envelopes.Accounts{
				accName: envelopes.Balance{currency: afterBal},
			},
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{currency: afterBal},
			},
		}

		actual := envelopes.CaluclateAmount(preOp, postOp)

		if !actual.Equal(expected) {
			t.Errorf("got:  %q\nwant: %q", actual, expected)
		}
	}
}

func renameBudget(t *testing.T) {
	const currency = "USD"
	const accName = "checking"
	accBalance := envelopes.Balance{currency: big.NewRat(100000, 1)}
	preOp := envelopes.State{
		Accounts: envelopes.Accounts{accName: accBalance},
		Budget: &envelopes.Budget{
			Children: map[string]*envelopes.Budget{
				"tweedleDee": {Balance: accBalance},
			},
		},
	}

	postOp := envelopes.State{
		Accounts: envelopes.Accounts{accName: accBalance},
		Budget: &envelopes.Budget{
			Children: map[string]*envelopes.Budget{
				"tweedleDum": {Balance: accBalance},
			},
		},
	}

	actual := envelopes.CaluclateAmount(preOp, postOp)
	expected := accBalance

	if !actual.Equal(expected) {
		t.Errorf("got:  %q\nwant: %q", actual, expected)
	}
}

func distributeFunds(t *testing.T) {
	const currency = "USD"
	const accName = "checking"
	accBalance := envelopes.Balance{currency: big.NewRat(1111, 1)}
	toDistribute := envelopes.Balance{currency: big.NewRat(1000, 1)}
	preOp := envelopes.State{
		Accounts: envelopes.Accounts{accName: accBalance},
		Budget: &envelopes.Budget{
			Children: map[string]*envelopes.Budget{
				"pocket":  {Balance: envelopes.Balance{currency: big.NewRat(1, 1)}},
				"grocery": {Balance: envelopes.Balance{currency: big.NewRat(10, 1)}},
				"savings": {Balance: envelopes.Balance{currency: big.NewRat(100, 1)}},
				"queue":   {Balance: toDistribute},
			},
		},
	}

	postOp := envelopes.State{
		Accounts: envelopes.Accounts{accName: accBalance},
		Budget: &envelopes.Budget{
			Children: map[string]*envelopes.Budget{
				"pocket":  {Balance: envelopes.Balance{currency: big.NewRat(51, 1)}},
				"grocery": {Balance: envelopes.Balance{currency: big.NewRat(460, 1)}},
				"savings": {Balance: envelopes.Balance{currency: big.NewRat(600, 1)}},
				"queue":   {Balance: envelopes.Balance{currency: big.NewRat(0, 1)}},
			},
		},
	}

	actual := envelopes.CaluclateAmount(preOp, postOp)
	expected := toDistribute

	if !actual.Equal(expected) {
		t.Errorf("got:  %q\nwant: %q", actual, expected)
	}
}

func threeWay(t *testing.T) {
	const currency = "USD"
	const accName = "checking"
	accBalance := envelopes.Balance{currency: big.NewRat(1111, 1)}

	preOp := envelopes.State{
		Accounts: envelopes.Accounts{accName: accBalance},
		Budget: &envelopes.Budget{
			Children: map[string]*envelopes.Budget{
				"pocket":  {Balance: envelopes.Balance{currency: big.NewRat(1, 1)}},
				"grocery": {Balance: envelopes.Balance{currency: big.NewRat(10, 1)}},
				"savings": {Balance: envelopes.Balance{currency: big.NewRat(100, 1)}},
			},
		},
	}

	postOp := envelopes.State{
		Accounts: envelopes.Accounts{accName: accBalance},
		Budget: &envelopes.Budget{
			Children: map[string]*envelopes.Budget{
				"pocket":  {Balance: envelopes.Balance{currency: big.NewRat(100, 1)}},
				"grocery": {Balance: envelopes.Balance{currency: big.NewRat(1, 1)}},
				"savings": {Balance: envelopes.Balance{currency: big.NewRat(10, 1)}},
			},
		},
	}

	actual := envelopes.CaluclateAmount(preOp, postOp)
	// Do I love this number, or think it's a good answer to this situation? No, not especially.
	// But capturing the behavior the current implementation has so that it doesn't accidentally
	// change is valuable. It's also good evidence for why tools built on this library might
	// want to allow users an escape hatch to override whatever this comes up with.
	expected := envelopes.Balance{currency: big.NewRat(99, 1)}

	if !actual.Equal(expected) {
		t.Errorf("got:  %q\nwant: %q", actual, expected)
	}
}
