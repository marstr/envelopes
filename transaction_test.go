package envelopes_test

import (
	"context"
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
				Expected: "30771be1c8910cffb0daf51c838ffddb82893612",
			},
			{
				Subject: envelopes.Transaction{
					Amount:   9807,
					Merchant: "Target",
					Time:     authorTime,
					Comment:  "Shoes",
					Parent:   envelopes.Transaction{}.ID(),
					State: &envelopes.State{
						Budget: &envelopes.Budget{
							Balance: 4511,
							Children: map[string]*envelopes.Budget{
								"grocery": {
									Balance: 6709,
								},
								"restaurants": {
									Balance: 12933,
								},
							},
						},
					},
				},
				Expected: "46426bfb73ea0c36c37e86a5fab064e1b686ed53",
			},
		}

		for _, tc := range testCases {
			got := tc.Subject.ID().String()
			if got != tc.Expected {
				t.Logf("got:  %q\nwant: %q", got, tc.Expected)
				t.Fail()
			}
		}
	}
}
