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
	"github.com/marstr/envelopes/persist"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"
)
import "github.com/marstr/envelopes"

func TestState_ID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	t.Run("deterministic", getTestStateIDDeterministic(ctx))
	t.Run("roundtrip", getTestStateIDRoundtrip(ctx))
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

func getTestStateIDRoundtrip(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		testCases := map[string]envelopes.State{
			"empty": {},
			"full": {
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(4590, 100)},
					"savings":  envelopes.Balance{"USD": big.NewRat(1, 100)},
				},
				Budget: &envelopes.Budget{
					Children: map[string]*envelopes.Budget{
						"foo": {
							Balance: envelopes.Balance{"USD": big.NewRat(2291, 100)},
						},
						"bar": {
							Balance: envelopes.Balance{"USD": big.NewRat(23, 1)},
						},
					},
				},
			},
			"accounts_only": {
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(4590, 100)},
					"savings":  envelopes.Balance{"USD": big.NewRat(1, 100)},
				},
			},
			"budget_only": {
				Budget: &envelopes.Budget{
					Children: map[string]*envelopes.Budget{
						"foo": {
							Balance: envelopes.Balance{"USD": big.NewRat(2291, 100)},
						},
						"bar": {
							Balance: envelopes.Balance{"USD": big.NewRat(2300, 100)},
						},
					},
				},
			},
		}

		tempLocation, err := ioutil.TempDir("", "envelopes_id_tests")
		if err != nil {
			t.Errorf("unable to create test location")
			return
		}
		defer os.RemoveAll(tempLocation)

		fs := persist.FileSystem{
			Root: tempLocation,
		}

		saver := persist.DefaultWriter{
			Stasher: fs,
		}
		reader := persist.DefaultLoader{
			Fetcher: fs,
		}

		for name, subject := range testCases {
			want := subject.ID()

			err := saver.Write(ctx, subject)
			if err != nil {
				t.Errorf("(%s) unable to write subject: %v", name, err)
				continue
			}

			var rehydrated envelopes.State
			err = reader.Load(ctx, want, &rehydrated)
			if err != nil {
				t.Errorf("(%s) unable to read subject: %v", name, err)
				continue
			}

			if got := rehydrated.ID(); !got.Equal(want) {
				t.Logf("\ngot: \t%s\nwant:\t%s", got, want)
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
					"savings": envelopes.Balance{"USD": big.NewRat(10000, 1)},
				},
			},
			Other: envelopes.State{
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(100, 1)},
					"savings": envelopes.Balance{"USD": big.NewRat(10000, 1)},
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

		if got.String() != want.String() {
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
