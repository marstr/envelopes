package persist_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

func TestMerge_Simple(t *testing.T) {
	const mainLine = "main"
	const feature = "alt"
	ctx := context.Background()
	var err error

	repo := persist.NewMockRepository(2, 4)

	initialBalances := envelopes.Transaction{
		State: &envelopes.State{
			Accounts: envelopes.Accounts{"checking": envelopes.Balance{"USD": big.NewRat(100, 1)}},
			Budget:   &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(100, 1)}},
		},
	}
	err = repo.WriteTransaction(ctx, initialBalances)
	if err != nil {
		t.Error(err)
		return
	}

	coffee := envelopes.Transaction{
		Amount: envelopes.Balance{"USD": big.NewRat(3, 1)},
		State: &envelopes.State{
			Accounts: envelopes.Accounts{"checking": envelopes.Balance{"USD": big.NewRat(97, 1)}},
			Budget:   &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(97, 1)}},
		},
		Parents: []envelopes.ID{initialBalances.ID()},
	}
	err = repo.WriteTransaction(ctx, coffee)
	if err != nil {
		t.Error(err)
		return
	}

	carwash := envelopes.Transaction{
		Amount: envelopes.Balance{"USD": big.NewRat(15, 1)},
		State: &envelopes.State{
			Accounts: envelopes.Accounts{"checking": envelopes.Balance{"USD": big.NewRat(85, 1)}},
			Budget:   &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(85, 1)}},
		},
		Parents: []envelopes.ID{initialBalances.ID()},
	}
	err = repo.WriteTransaction(ctx, carwash)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.WriteBranch(ctx, mainLine, coffee.ID())
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.WriteBranch(ctx, feature, carwash.ID())
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.SetCurrent(ctx, mainLine)
	if err != nil {
		t.Error(err)
		return
	}

	var got envelopes.State
	var conflicts []persist.Conflict
	got, conflicts, err = persist.Merge(ctx, repo, []persist.RefSpec{mainLine, feature})
	if err != nil {
		fmt.Println("Failed: ", err)
		return
	}

	expected := envelopes.Balance{"USD": big.NewRat(82, 1)}
	if !got.Accounts["checking"].Equal(expected) {
		t.Errorf("incorrect account balance\n\tgot:  %s\n\twant: %s", got, expected)
	}

	if !got.Budget.Balance.Equal(expected) {
		t.Errorf("incorrect budget balance\n\tgot:  %s\n\twant: %s", got, expected)
	}

	if len(conflicts) != 0 {
		t.Errorf("unexpected conflicts: %v", len(conflicts))
	}
}

func TestMerge_Threeway(t *testing.T) {
	const mainLine = "main"
	const feature1 = "alt1"
	const feature2 = "alt2"

	ctx := context.Background()
	var err error

	repo := persist.NewMockRepository(3, 5)

	initialBalances := envelopes.Transaction{
		State: &envelopes.State{
			Accounts: envelopes.Accounts{"checking": envelopes.Balance{"USD": big.NewRat(100, 1)}},
			Budget:   &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(100, 1)}},
		},
	}
	err = repo.WriteTransaction(ctx, initialBalances)
	if err != nil {
		t.Error(err)
		return
	}

	coffee := envelopes.Transaction{
		Amount: envelopes.Balance{"USD": big.NewRat(3, 1)},
		State: &envelopes.State{
			Accounts: envelopes.Accounts{"checking": envelopes.Balance{"USD": big.NewRat(97, 1)}},
			Budget:   &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(97, 1)}},
		},
		Parents: []envelopes.ID{initialBalances.ID()},
	}
	err = repo.WriteTransaction(ctx, coffee)
	if err != nil {
		t.Error(err)
		return
	}

	carwash := envelopes.Transaction{
		Amount: envelopes.Balance{"USD": big.NewRat(15, 1)},
		State: &envelopes.State{
			Accounts: envelopes.Accounts{"checking": envelopes.Balance{"USD": big.NewRat(85, 1)}},
			Budget:   &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(85, 1)}},
		},
		Parents: []envelopes.ID{initialBalances.ID()},
	}
	err = repo.WriteTransaction(ctx, carwash)
	if err != nil {
		t.Error(err)
		return
	}

	videogame := envelopes.Transaction{
		Amount: envelopes.Balance{"USD": big.NewRat(60, 1)},
		State: &envelopes.State{
			Accounts: envelopes.Accounts{"checking": envelopes.Balance{"USD": big.NewRat(40, 1)}},
			Budget:   &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(40, 1)}},
		},
		Parents: []envelopes.ID{initialBalances.ID()},
	}
	err = repo.WriteTransaction(ctx, videogame)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.WriteBranch(ctx, mainLine, coffee.ID())
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.WriteBranch(ctx, feature1, carwash.ID())
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.WriteBranch(ctx, feature2, videogame.ID())
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.SetCurrent(ctx, mainLine)
	if err != nil {
		t.Error(err)
		return
	}

	var got envelopes.State
	var conflicts []persist.Conflict
	got, conflicts, err = persist.Merge(ctx, repo, []persist.RefSpec{mainLine, feature1, feature2})
	if err != nil {
		fmt.Println("Failed: ", err)
		return
	}

	expected := envelopes.Balance{"USD": big.NewRat(22, 1)}
	if !got.Accounts["checking"].Equal(expected) {
		t.Errorf("incorrect account balance\n\tgot:  %s\n\twant: %s", got, expected)
	}

	if !got.Budget.Balance.Equal(expected) {
		t.Errorf("incorrect budget balance\n\tgot:  %s\n\twant: %s", got, expected)
	}

	if len(conflicts) != 0 {
		t.Errorf("unexpected conflicts: %v", len(conflicts))
	}
}
