package persist

import (
	"context"
	"math/big"
	"testing"

	"github.com/marstr/envelopes"
)

func TestCache_Load(t *testing.T) {
	var ctx context.Context
	deadline, ok := t.Deadline()
	if ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	t.Run("UsePassThroughOnMiss", testUsePassThroughOnMiss(ctx))
	t.Run("ModifiedTransactionOnHit", testModifiedTransactionOnHit(ctx))
	t.Run("ModifiedTransactionOnMiss", testModifiedTransactionOnMiss(ctx))
	t.Run("ModifiedStateOnMiss", testModifiedStateOnMiss(ctx))
	t.Run("ModifiedAccountOnMiss", testModifiedAccountOnMiss(ctx))
	t.Run("ModifiedBudgetOnMiss", testModifiedBudgetOnMiss(ctx))
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
		err := subject.LoadState(ctx, want.ID(), &got)
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

func testModifiedTransactionOnHit(ctx context.Context) func(t *testing.T) {
	return func(t *testing.T) {
		subject := NewCache(10)
		const expectedMerchant = "EXL Auto Detail"
		toWrite := envelopes.Transaction{
			Merchant: expectedMerchant,
		}
		writtenId := toWrite.ID()

		err := subject.WriteTransaction(ctx, toWrite)
		if err != nil {
			t.Error(err)
			return
		}

		var firstLoad envelopes.Transaction

		err = subject.LoadTransaction(ctx, writtenId, &firstLoad)
		if err != nil {
			t.Error(err)
			return
		}

		if firstLoad.Merchant != expectedMerchant {
			t.Errorf("unexpected merchant on first load\n\tgot:  %q\n\twant: %q", firstLoad.Merchant, expectedMerchant)
		}

		firstLoad.Merchant = "Cedar Rapids Auto Detail"

		var secondLoad envelopes.Transaction
		err = subject.LoadTransaction(ctx, writtenId, &secondLoad)
		if err != nil {
			t.Error(err)
			return
		}

		if secondLoad.Merchant != expectedMerchant {
			t.Errorf("unexpected merchant on second load\n\tgot:  %q\n\twant: %q", secondLoad.Merchant, expectedMerchant)
		}
	}
}

func testModifiedTransactionOnMiss(ctx context.Context) func(t *testing.T) {
	return func(t *testing.T) {
		underlyer := NewCache(10)
		subject := NewCache(10)
		subject.Loader = underlyer

		const expectedMerchant = "EXL Auto Detail"
		toWrite := envelopes.Transaction{
			Merchant: expectedMerchant,
		}
		writtenId := toWrite.ID()

		err := underlyer.WriteTransaction(ctx, toWrite)
		if err != nil {
			t.Error(err)
			return
		}

		var firstLoad envelopes.Transaction

		err = subject.LoadTransaction(ctx, writtenId, &firstLoad)
		if err != nil {
			t.Error(err)
			return
		}

		if firstLoad.Merchant != expectedMerchant {
			t.Errorf("unexpected merchant on first load\n\tgot:  %q\n\twant: %q", firstLoad.Merchant, expectedMerchant)
		}

		firstLoad.Merchant = "Cedar Rapids Auto Detail"

		var secondLoad envelopes.Transaction
		err = subject.LoadTransaction(ctx, writtenId, &secondLoad)
		if err != nil {
			t.Error(err)
			return
		}

		if secondLoad.Merchant != expectedMerchant {
			t.Errorf("unexpected merchant on second load\n\tgot:  %q\n\twant: %q", secondLoad.Merchant, expectedMerchant)
		}
	}
}

func testModifiedStateOnMiss(ctx context.Context) func(t *testing.T) {
	return func(t *testing.T) {
		underlyer := NewCache(10)
		subject := NewCache(10)
		subject.Loader = underlyer

		var expectedBudget = envelopes.Budget{
			Balance: envelopes.Balance{"USD": big.NewRat(100, 1)},
		}
		toWrite := envelopes.State{
			Budget: &expectedBudget,
		}
		writtenId := toWrite.ID()

		err := underlyer.WriteState(ctx, toWrite)
		if err != nil {
			t.Error(err)
			return
		}

		var firstLoad envelopes.State

		err = subject.LoadState(ctx, writtenId, &firstLoad)
		if err != nil {
			t.Error(err)
			return
		}

		if !firstLoad.Budget.Equal(expectedBudget) {
			t.Errorf("unexpected balance on first load\n\tgot:  %v\n\twant: %v", firstLoad.Budget, expectedBudget)
		}

		firstLoad.Budget = &envelopes.Budget{
			Balance: envelopes.Balance{"EUR": big.NewRat(45, 1)},
		}

		var secondLoad envelopes.State
		err = subject.LoadState(ctx, writtenId, &secondLoad)
		if err != nil {
			t.Error(err)
			return
		}

		if !secondLoad.Budget.Equal(expectedBudget) {
			t.Errorf("unexpected balance on second load\n\tgot:  %v\n\twant: %v", secondLoad.Budget.Balance, expectedBudget)
		}
	}
}

func testModifiedAccountOnMiss(ctx context.Context) func(t *testing.T) {
	const accountName = "checking"
	return func(t *testing.T) {
		underlyer := NewCache(10)
		subject := NewCache(10)
		subject.Loader = underlyer

		var expectedBalance = envelopes.Balance{"USD": big.NewRat(100, 1)}
		toWrite := envelopes.Accounts{
			accountName: expectedBalance,
		}
		writtenId := toWrite.ID()

		err := underlyer.WriteAccounts(ctx, toWrite)
		if err != nil {
			t.Error(err)
			return
		}

		var firstLoad envelopes.Accounts

		err = subject.LoadAccounts(ctx, writtenId, &firstLoad)
		if err != nil {
			t.Error(err)
			return
		}

		if !firstLoad[accountName].Equal(expectedBalance) {
			t.Errorf("unexpected balance on first load\n\tgot:  %q\n\twant: %q", firstLoad[accountName], expectedBalance)
		}

		firstLoad[accountName] = envelopes.Balance{"EUR": big.NewRat(45, 1)}

		var secondLoad envelopes.Accounts
		err = subject.LoadAccounts(ctx, writtenId, &secondLoad)
		if err != nil {
			t.Error(err)
			return
		}

		if !secondLoad[accountName].Equal(expectedBalance) {
			t.Errorf("unexpected balance on second load\n\tgot:  %q\n\twant: %q", secondLoad[accountName], expectedBalance)
		}
	}
}

func testModifiedBudgetOnMiss(ctx context.Context) func(t *testing.T) {
	return func(t *testing.T) {
		underlyer := NewCache(10)
		subject := NewCache(10)
		subject.Loader = underlyer

		var expectedBalance = envelopes.Balance{"USD": big.NewRat(100, 1)}
		toWrite := envelopes.Budget{Balance: expectedBalance}
		writtenId := toWrite.ID()

		err := underlyer.WriteBudget(ctx, toWrite)
		if err != nil {
			t.Error(err)
			return
		}

		var firstLoad envelopes.Budget

		err = subject.LoadBudget(ctx, writtenId, &firstLoad)
		if err != nil {
			t.Error(err)
			return
		}

		if !firstLoad.Balance.Equal(expectedBalance) {
			t.Errorf("unexpected balance on first load\n\tgot:  %q\n\twant: %q", firstLoad.Balance, expectedBalance)
		}

		firstLoad.Balance = envelopes.Balance{"EUR": big.NewRat(45, 1)}

		var secondLoad envelopes.Budget
		err = subject.LoadBudget(ctx, writtenId, &secondLoad)
		if err != nil {
			t.Error(err)
			return
		}

		if !secondLoad.Balance.Equal(expectedBalance) {
			t.Errorf("unexpected balance on second load\n\tgot:  %q\n\twant: %q", secondLoad.Balance, expectedBalance)
		}
	}
}
