package persist

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/marstr/envelopes"
)

func TestCache_Load(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	t.Run("UsePassThroughOnMiss", testUsePassThroughOnMiss(ctx))
}

func testUsePassThroughOnMiss(ctx context.Context) func(t *testing.T) {
	return func(t *testing.T) {
		want := envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{"MSFT": big.NewRat(24, 1)},
				Children: map[string]*envelopes.Budget{
					"foo": {
						Balance: envelopes.Balance{"TMUS": big.NewRat(35, 1)},
					},
				},
			},
			Accounts: map[string]envelopes.Balance{
				"brokerage": {
					"MSFT": big.NewRat(24, 1),
					"TMUS": big.NewRat(35, 1),
				},
			},
		}

		passThrough := NewCache(10)
		if err := passThrough.WriteState(ctx, want); err != nil {
			t.Error(err)
			return
		}

		subject := NewCache(10)
		subject.Loader = passThrough

		var got envelopes.State
		err := subject.Load(ctx, want.ID(), &got)
		if err != nil {
			t.Error(err)
			return
		}

		if !got.Equal(want) {
			t.Logf("got: %s, want: %s", got.String(), want.String())
			t.Fail()
		}
	}
}
