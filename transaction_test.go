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
	t.Run("recordIdIncluded", getEnsureBankIdIncluded(ctx))
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
				Expected: "48087310cd4fafe77745abccea008388ec10775c",
			},
			{
				Subject: envelopes.Transaction{
					Amount:     envelopes.Balance{"USD": big.NewRat(9807, 100)},
					Merchant:   "Target",
					PostedTime: authorTime,
					Comment:    "Shoes",
					Parents: []envelopes.ID{
						envelopes.Transaction{}.ID(),
					},
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
				Expected: "71e6c71d9acdbe69598aceef7c703cdb86f563c1",
			},
			{
				Subject: envelopes.Transaction{
					State: &envelopes.State{
						Budget: &envelopes.Budget{
							Balance: envelopes.Balance{"USD": big.NewRat(10005, 100)},
							Children: map[string]*envelopes.Budget{
								"grocery": {
									Balance: envelopes.Balance{"USD": big.NewRat(5000, 100)},
								},
							},
						},
						Accounts: envelopes.Accounts{
							"checking": envelopes.Balance{},
						},
					},
					ActualTime:  authorTime.Add(-40 * time.Minute),
					EnteredTime: authorTime,
					Amount:      envelopes.Balance{"LAT": big.NewRat(10, 1)},
					Merchant:    "Quark",
					Committer: envelopes.User{
						FullName: "James Strobel",
						Email:    "jamesthebrother@yahoo.org", // Need I point out this is not a real email?
					},
					Comment: "Undisclosed Ferengi wares... totally above board",
					Parents: []envelopes.ID{
						envelopes.Transaction{ActualTime: authorTime.Add(-2 * time.Hour)}.ID(),
					},
				},
				Expected: "7277fbcedec2739dec52b6fd158d014d8209c9eb",
			},
			{
				Subject: envelopes.Transaction{
					Amount:     envelopes.Balance{"USD": big.NewRat(9807, 100)},
					Merchant:   "Target",
					PostedTime: authorTime,
					Comment:    "Shoes",
					Parents: []envelopes.ID{
						envelopes.Transaction{}.ID(),
					},
					RecordID: "20201212 575073 2,000 202,012,128,756",
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
						Accounts: envelopes.Accounts{
							"checking": envelopes.Balance{"USD": big.NewRat(24153, 100)},
						},
					},
				},
				Expected: "ebad5097e0013577f9926fa2f5f5bec385608794",
			},
			{
				Subject: envelopes.Transaction{
					Amount:     envelopes.Balance{"USD": big.NewRat(9807, 100)},
					Merchant:   "Target",
					PostedTime: authorTime,
					Comment:    "Shoes",
					Parents: []envelopes.ID{
						envelopes.Transaction{}.ID(),
					},
					RecordID: "20201212 575073 2,000 202,012,128,756",
					Reverts:  envelopes.Transaction{Comment: "Clearly erroneous"}.ID(),
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
						Accounts: envelopes.Accounts{
							"checking": envelopes.Balance{"USD": big.NewRat(24153, 100)},
						},
					},
				},
				Expected: "0d186bf4bc88d299b483c3d3d8e2b0147529e50e",
			},
		}

		for _, tc := range testCases {
			select {
			case <-ctx.Done():
				t.Error(ctx.Err())
				return
			default:
				// Intentionally Left Blank
			}

			got := tc.Subject.ID().String()
			if got != tc.Expected {
				t.Logf("\ngot:  %q\nwant: %q", got, tc.Expected)
				t.Fail()
			}
		}
	}
}

func getEnsureBankIdIncluded(_ context.Context) func(*testing.T) {
	authorTime, err := time.Parse(time.RFC3339, "2019-10-25T16:23:48-07:00")
	if err != nil {
		panic(err)
	}
	return func(t *testing.T) {
		with := envelopes.Transaction{
			Amount:     envelopes.Balance{"USD": big.NewRat(9807, 100)},
			Merchant:   "Target",
			PostedTime: authorTime,
			Comment:    "Shoes",
			Parents: []envelopes.ID{
				envelopes.Transaction{}.ID(),
			},
			RecordID: "20201212 575073 2,000 202,012,128,756",
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
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(24153, 100)},
				},
			},
		}

		without := with
		without.RecordID = ""

		if with.ID() == without.ID() {
			t.Logf("including a bank record ID should change the ID of the transaction")
			t.Fail()
		}
	}
}
