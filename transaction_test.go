package envelopes_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/marstr/envelopes"
)

func TestTransaction_ID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	t.Run("lock", getTestTransactionIDLock(ctx))
}

func getTestTransactionIDLock(ctx context.Context) func(*testing.T) {
	authorTime, err := time.Parse(time.RFC3339, "2019-10-25T16:23:48-07:00")
	if err != nil {
		panic(err)
	}

	return func(t *testing.T) {
		testCases := []struct {
			Subject  envelopes.Transaction
			Expected string
		}{
			{
				Subject:  envelopes.Transaction{},
				Expected: "9273fa1765fd735527196071b244bbafa5706ddf",
			},
			{
				Subject: envelopes.Transaction{
					Amount:     envelopes.Balance{"USD": big.NewRat(9807, 100)},
					Merchant:   "Target",
					PostedTime: authorTime,
					Comment:    "Shoes",
					Parent:     envelopes.Transaction{}.ID(),
					State: &envelopes.State{
						Budget: &envelopes.Budget{
							Balance: envelopes.Balance{"USD": big.NewRat(4511, 100)},
							Children: map[string]*envelopes.Budget{
								"grocery": {
									Balance: envelopes.Balance{"USD": big.NewRat(6709, 100)},
								},
								"restaurants": {
									Balance: envelopes.Balance{"USD": big.NewRat(12933, 100)},
								},
							},
						},
					},
				},
				Expected: "c8a29d9bc845908238caf17ee0629e4bdeabb2ef",
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
