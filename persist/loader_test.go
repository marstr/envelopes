package persist

import (
	"context"
	"math/big"
	"testing"

	"github.com/marstr/envelopes"
)

func TestNearestCommonAncestor_sharedParent(t *testing.T) {
	var err error
	ctx := context.Background()
	root := envelopes.Transaction{Comment: "Root"}
	rid := root.ID()
	gen1a := envelopes.Transaction{Comment: "Gen1 - A", Parents: []envelopes.ID{rid}}
	gen1aid := gen1a.ID()
	gen1b := envelopes.Transaction{Comment: "Gen1 - B", Parents: []envelopes.ID{rid}}
	gen1bid := gen1b.ID()

	repo := NewMockRepository(0, 4)
	err = repo.Write(ctx, root)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.Write(ctx, gen1a)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.Write(ctx, gen1b)
	if err != nil {
		t.Error(err)
		return
	}

	var got envelopes.ID
	got, err = NearestCommonAncestor(ctx, repo, gen1aid, gen1bid)
	if err != nil {
		t.Error(err)
		return
	}

	if !got.Equal(rid) {
		t.Errorf("got: %s want: %s", got, rid)
	}
}

func TestNearestCommonAncestor_parentAndChild(t *testing.T) {
	var err error
	ctx := context.Background()
	parent := envelopes.Transaction{Comment: "Parent"}
	pid := parent.ID()
	child := envelopes.Transaction{Comment: "Child", Parents: []envelopes.ID{pid}}
	cid := child.ID()

	repo := NewMockRepository(0, 2)
	err = repo.Write(ctx, parent)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.Write(ctx, child)
	if err != nil {
		t.Error(err)
		return
	}

	var got envelopes.ID
	got, err = NearestCommonAncestor(ctx, repo, pid, cid)
	if err != nil {
		t.Error(err)
		return
	}

	if !got.Equal(pid) {
		t.Errorf("got: %s want: %s", got, pid)
	}
}

func TestNearestCommonAncestor_multipleYs(t *testing.T) {
	var err error
	ctx := context.Background()
	root := envelopes.Transaction{Comment: "Root"}
	rid := root.ID()
	gen1a := envelopes.Transaction{Comment: "Gen1 - A", Parents: []envelopes.ID{rid}}
	gen1aid := gen1a.ID()
	gen1b := envelopes.Transaction{Comment: "Gen1 - B", Parents: []envelopes.ID{rid}}
	gen1bid := gen1b.ID()

	gen2aa := envelopes.Transaction{Comment: "Gen2 - AA", Parents: []envelopes.ID{gen1aid}}
	gen2aaid := gen2aa.ID()
	gen2ab := envelopes.Transaction{Comment: "Gen2 - AB", Parents: []envelopes.ID{gen1aid}}
	gen2abid := gen2ab.ID()
	gen2ba := envelopes.Transaction{Comment: "Gen2 - BA", Parents: []envelopes.ID{gen1bid}}
	gen2baid := gen2ba.ID()
	gen2bb := envelopes.Transaction{Comment: "Gen2 - BB", Parents: []envelopes.ID{gen1bid}}
	gen2bbid := gen2bb.ID()

	gen3a := envelopes.Transaction{Comment: "Gen3 - A", Parents: []envelopes.ID{gen2aaid, gen2abid}}
	gen3aid := gen3a.ID()
	gen3b := envelopes.Transaction{Comment: "Gen3 - B", Parents: []envelopes.ID{gen2baid, gen2bbid}}
	gen3bid := gen3b.ID()


	repo := NewMockRepository(0, 9)
	err = repo.Write(ctx, root)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.Write(ctx, gen1a)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.Write(ctx, gen1b)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.Write(ctx, gen2aa)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.Write(ctx, gen2ab)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.Write(ctx, gen2ba)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.Write(ctx, gen2bb)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.Write(ctx, gen3a)
	if err != nil {
		t.Error(err)
		return
	}


	err = repo.Write(ctx, gen3b)
	if err != nil {
		t.Error(err)
		return
	}

	var got envelopes.ID
	got, err = NearestCommonAncestor(ctx, repo, gen3aid, gen3bid)
	if err != nil {
		t.Error(err)
		return
	}

	if !got.Equal(rid) {
		t.Errorf("got: %s want: %s", got, rid)
	}
}

func TestLoadImpact_simple(t *testing.T) {
	ctx := context.Background()
	var err error
	parent := envelopes.Transaction{
		Comment: "Parent",
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{
					"USD": big.NewRat(10,1),
				},
			},
			Accounts: map[string]envelopes.Balance{
				"checking": {
					"USD":big.NewRat(10,1),
				},
			},
		},
	}
	pid := parent.ID()

	child := envelopes.Transaction{
		Comment: "Child",
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{
					"USD": big.NewRat(15,1),
				},
			},
			Accounts: map[string]envelopes.Balance{
				"checking": {
					"USD":big.NewRat(15,1),
				},
			},
		},
		Parents: []envelopes.ID{pid},
	}

	repo := NewMockRepository(0, 2)
	err = repo.Write(ctx, parent)
	if err != nil {
		t.Error(err)
		return
	}
	err = repo.Write(ctx, child)
	if err != nil {
		t.Error(err)
		return
	}

	var got envelopes.Impact
	got, err = LoadImpact(ctx, repo, child)
	if err != nil {
		t.Error(err)
		return
	}

	want := envelopes.Impact{
		Budget: &envelopes.Budget{
			Balance: envelopes.Balance{
				"USD":big.NewRat(5,1),
			},
		},
		Accounts: map[string]envelopes.Balance{
			"checking":{
				"USD": big.NewRat(5,1),
			},
		},
	}

	if !got.Equal(want) {
		t.Errorf("got:  %v\nwant: %v\n", got, want)
	}
}

func TestLoadImpact_noImpactMerge(t *testing.T) {
	ctx := context.Background()
	var err error
	gen1 := envelopes.Transaction{
		Comment: "Gen 1",
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{
					"USD": big.NewRat(30,1),
				},
			},
			Accounts: map[string]envelopes.Balance{
				"checking": {
					"USD":big.NewRat(10,1),
				},
				"savings": {
					"USD":big.NewRat(20, 1),
				},
			},
		},
	}
	gen1id := gen1.ID()

	gen2a := envelopes.Transaction{
		Comment: "Gen 2 - a",
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{
					"USD": big.NewRat(35,1),
				},
			},
			Accounts: map[string]envelopes.Balance{
				"checking": {
					"USD":big.NewRat(15,1),
				},
				"savings": {
					"USD":big.NewRat(20,1),
				},
			},
		},
		Parents: []envelopes.ID{gen1id},
	}

	gen2b := envelopes.Transaction{
		Comment: "Gen 2 - b",
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{
					"USD": big.NewRat(35,1),
				},
			},
			Accounts: map[string]envelopes.Balance{
				"checking": {
					"USD":big.NewRat(10,1),
				},
				"savings": {
					"USD":big.NewRat(25,1),
				},
			},
		},
		Parents: []envelopes.ID{gen1id},
	}

	gen3 := envelopes.Transaction{
		Comment: "Gen 3",
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{
					"USD": big.NewRat(40,1),
				},
			},
			Accounts: map[string]envelopes.Balance{
				"checking": {
					"USD":big.NewRat(15,1),
				},
				"savings": {
					"USD":big.NewRat(25,1),
				},
			},
		},
		Parents: []envelopes.ID{gen2a.ID(), gen2b.ID()},
	}

	repo := NewMockRepository(0, 4)
	err = repo.Write(ctx, gen1)
	if err != nil {
		t.Error(err)
		return
	}
	err = repo.Write(ctx, gen2a)
	if err != nil {
		t.Error(err)
		return
	}
	err = repo.Write(ctx, gen2b)
	if err != nil {
		t.Error(err)
		return
	}
	err = repo.Write(ctx, gen3)
	if err != nil {
		t.Error(err)
		return
	}

	var got envelopes.Impact
	got, err = LoadImpact(ctx, repo, gen3)
	if err != nil {
		t.Error(err)
		return
	}

	want := envelopes.Impact{}

	if !got.Equal(want) {
		t.Errorf("got:  %v\nwant: %v\n", got, want)
	}
}

func TestLoadImpact_duplicateReconciled(t *testing.T) {
	ctx := context.Background()
	var err error
	gen1 := envelopes.Transaction{
		Comment: "Gen 1",
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{
					"USD": big.NewRat(30,1),
				},
			},
			Accounts: map[string]envelopes.Balance{
				"checking": {
					"USD":big.NewRat(10,1),
				},
				"savings": {
					"USD":big.NewRat(20, 1),
				},
			},
		},
	}
	gen1id := gen1.ID()

	gen2a := envelopes.Transaction{
		Comment: "Gen 2 - a",
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{
					"USD": big.NewRat(35,1),
				},
			},
			Accounts: map[string]envelopes.Balance{
				"checking": {
					"USD":big.NewRat(15,1),
				},
				"savings": {
					"USD":big.NewRat(20,1),
				},
			},
		},
		Parents: []envelopes.ID{gen1id},
	}

	gen2b := envelopes.Transaction{
		Comment: "Gen 2 - b",
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{
					"USD": big.NewRat(35,1),
				},
			},
			Accounts: map[string]envelopes.Balance{
				"checking": {
					"USD":big.NewRat(15,1),
				},
				"savings": {
					"USD":big.NewRat(20,1),
				},
			},
		},
		Parents: []envelopes.ID{gen1id},
	}

	gen3 := envelopes.Transaction{
		Comment: "Gen 3",
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{
					"USD": big.NewRat(35,1),
				},
			},
			Accounts: map[string]envelopes.Balance{
				"checking": {
					"USD":big.NewRat(15,1),
				},
				"savings": {
					"USD":big.NewRat(20,1),
				},
			},
		},
		Parents: []envelopes.ID{gen2a.ID(), gen2b.ID()},
	}

	repo := NewMockRepository(0, 4)
	err = repo.Write(ctx, gen1)
	if err != nil {
		t.Error(err)
		return
	}
	err = repo.Write(ctx, gen2a)
	if err != nil {
		t.Error(err)
		return
	}
	err = repo.Write(ctx, gen2b)
	if err != nil {
		t.Error(err)
		return
	}
	err = repo.Write(ctx, gen3)
	if err != nil {
		t.Error(err)
		return
	}

	var got envelopes.Impact
	got, err = LoadImpact(ctx, repo, gen3)
	if err != nil {
		t.Error(err)
		return
	}

	want := envelopes.Impact{
		Budget: &envelopes.Budget{
			Balance: envelopes.Balance{
				"USD": big.NewRat(-5,1),
			},
		},
		Accounts: map[string]envelopes.Balance{
			"checking": {
				"USD":big.NewRat(-5,1),
			},
		},
	}

	if !got.Equal(want) {
		t.Errorf("got:  %v\nwant: %v\n", got, want)
	}
}