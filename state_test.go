package envelopes_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/marstr/envelopes"
)

func TestState_MarshalJSON(t *testing.T) {
	testCases := []struct {
		envelopes.State
		want string
	}{
		{
			envelopes.State{},
			`{"budget":{"name":"","balance":0},"accounts":null,"parent":"0000000000000000000000000000000000000000"}`,
		},
		{
			envelopes.State{
				Budget: envelopes.Budget{
					Name:    "Budget1",
					Balance: 239177,
					Children: []envelopes.Budget{
						{
							Name:    "Child1",
							Balance: 4490,
						},
						{
							Name:    "Child2",
							Balance: 9997,
						},
					},
				},
				Accounts: []envelopes.Account{
					{
						Name:    "Account1",
						Balance: 252664,
					},
					{
						Name:    "Account2",
						Balance: 1000,
					},
				},
				Parent: envelopes.ID{21, 44, 243, 88, 172, 90, 67, 00, 99, 43, 11, 67, 89, 17, 244, 61, 55, 44, 80, 98},
			},
			`{"budget":{"name":"Budget1","balance":239177,"children":[{"name":"Child1","balance":4490},{"name":"Child2","balance":9997}]},"accounts":[{"name":"Account1","balance":252664},{"name":"Account2","balance":1000}],"parent":"152cf358ac5a4300632b0b435911f43d372c5062"}`,
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			if got, err := json.Marshal(tc.State); err != nil {
				t.Error(err)
			} else if cast := string(got); cast != tc.want {
				t.Logf("\ngot:  %q\nwant: %q", cast, tc.want)
				t.Fail()
			}
		})
	}
}

func TestState_ID(t *testing.T) {
	testCases := []struct {
		envelopes.State
		expected [20]byte
	}{
		// Calculated here: https://play.golang.org/p/r_q6EAZ-MT
		{
			envelopes.State{},
			envelopes.ID{41, 249, 117, 10, 100, 40, 62, 194, 174, 210, 230, 58, 103, 204, 146, 64, 123, 79, 180, 179},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprint(tc.State), func(t *testing.T) {
			if got := tc.ID(); got != tc.expected {
				t.Logf("\ngot:  %x\nwant: %x", got, tc.expected)
				t.Fail()
			}
		})
	}
}
