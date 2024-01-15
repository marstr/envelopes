package persist

import (
	"context"
	"testing"

	"github.com/marstr/envelopes"
)

func TestWalker_Walk(t *testing.T) {
	ctx := context.Background()
	t.Run("single line", chain(ctx))
	t.Run("basic fork", fork(ctx))
	t.Run("skip when told", respectSkipAncestors(ctx))
	t.Run("respect depth", respectDepth(ctx))
}

func chain(ctx context.Context) func(t *testing.T) {
	cache := NewCache(2)
	a := envelopes.Transaction{
		Comment: "First!",
	}
	aid := a.ID()
	err := cache.WriteTransaction(ctx, a)
	if err != nil {
		panic(err)
	}

	b := envelopes.Transaction{
		Comment: "Second!",
		Parents: []envelopes.ID{
			aid,
		},
	}
	bid := b.ID()
	err = cache.WriteTransaction(ctx, b)
	if err != nil {
		panic(err)
	}

	return func(t *testing.T) {
		expected := map[envelopes.ID]struct{}{
			aid: {},
			bid: {},
		}

		texasRanger := Walker{
			Loader: cache,
		}

		err := texasRanger.Walk(ctx, func(ctx context.Context, currentId envelopes.ID, transaction envelopes.Transaction) error {
			_, ok := expected[currentId]
			if ok {
				delete(expected, currentId)
			} else {
				t.Errorf("unexpected transaction ID: %s", currentId)
			}
			return nil
		}, bid)

		if err != nil {
			t.Error(err)
		}

		if len(expected) != 0 {
			t.Error("didn't see expected transactions")
		}
	}
}

func fork(ctx context.Context) func(t *testing.T) {
	cache := NewCache(3)
	a := envelopes.Transaction{
		Comment: "First!",
	}
	aid := a.ID()
	err := cache.WriteTransaction(ctx, a)
	if err != nil {
		panic(err)
	}

	b := envelopes.Transaction{
		Comment: "Second!",
		Parents: []envelopes.ID{
			aid,
		},
	}
	bid := b.ID()
	err = cache.WriteTransaction(ctx, b)
	if err != nil {
		panic(err)
	}

	c := envelopes.Transaction{
		Comment: "Third!",
		Parents: []envelopes.ID{
			aid,
		},
	}
	cid := c.ID() // Eastern Iowa Airport shout-out
	err = cache.WriteTransaction(ctx, c)
	if err != nil {
		panic(err)
	}

	return func(t *testing.T) {
		expected := map[envelopes.ID]struct{}{
			aid: {},
			bid: {},
			cid: {},
		}
		t.Logf("a: %s", aid)
		t.Logf("b: %s", bid)
		t.Logf("c: %s", cid)

		texasRanger := Walker{
			Loader: cache,
		}

		err := texasRanger.Walk(ctx, func(ctx context.Context, currentId envelopes.ID, transaction envelopes.Transaction) error {
			_, ok := expected[currentId]
			if ok {
				delete(expected, currentId)
			} else {
				t.Errorf("unexpected transaction ID: %s", currentId)
			}
			return nil
		}, bid, cid)

		if err != nil {
			t.Error(err)
		}

		if len(expected) != 0 {
			t.Error("didn't see expected transactions")
		}
	}
}

func respectSkipAncestors(ctx context.Context) func(t *testing.T) {
	cache := NewCache(4)
	a := envelopes.Transaction{
		Comment: "First!",
	}
	aid := a.ID()
	err := cache.WriteTransaction(ctx, a)
	if err != nil {
		panic(err)
	}

	b := envelopes.Transaction{
		Comment: "Second!",
		Parents: []envelopes.ID{
			aid,
		},
	}
	bid := b.ID()
	err = cache.WriteTransaction(ctx, b)
	if err != nil {
		panic(err)
	}

	c := envelopes.Transaction{
		Comment: "Third!",
		Parents: []envelopes.ID{
			bid,
		},
	}
	cid := c.ID() // Eastern Iowa Airport shout-out
	err = cache.WriteTransaction(ctx, c)
	if err != nil {
		panic(err)
	}

	d := envelopes.Transaction{
		Comment: "Fourth!",
		Parents: []envelopes.ID{
			cid,
		},
	}
	did := d.ID()
	err = cache.WriteTransaction(ctx, d)
	if err != nil {
		panic(err)
	}

	return func(t *testing.T) {
		expected := map[envelopes.ID]struct{}{
			cid: {},
			did: {},
		}
		t.Logf("a: %s", aid)
		t.Logf("b: %s", bid)
		t.Logf("c: %s", cid)
		t.Logf("d: %s", did)

		texasRanger := Walker{
			Loader: cache,
		}

		err := texasRanger.Walk(ctx, func(ctx context.Context, currentId envelopes.ID, transaction envelopes.Transaction) error {
			_, ok := expected[currentId]

			if ok {
				delete(expected, currentId)
			} else {
				t.Errorf("unexpected transaction ID: %s", currentId)
			}

			if currentId.Equal(cid) {
				return ErrSkipAncestors{}
			}

			return nil
		}, did)

		if err != nil {
			t.Error(err)
		}

		if len(expected) != 0 {
			t.Error("didn't see expected transactions")
		}
	}
}

func respectDepth(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		repo := NewMockRepository(0, 4)

		gen1 := envelopes.Transaction{Comment: "Gen 1"}
		gen1id := gen1.ID()
		if err := repo.WriteTransaction(ctx, gen1); err != nil {
			t.Error(err)
			return
		}

		gen2a := envelopes.Transaction{Comment: "Gen 2a", Parents: []envelopes.ID{gen1id}}
		gen2aid := gen2a.ID()
		if err := repo.WriteTransaction(ctx, gen2a); err != nil {
			t.Error(err)
			return
		}

		gen2b := envelopes.Transaction{Comment: "Gen 2b", Parents: []envelopes.ID{gen1id}}
		gen2bid := gen2b.ID()
		if err := repo.WriteTransaction(ctx, gen2b); err != nil {
			t.Error(err)
			return
		}

		gen3 := envelopes.Transaction{Comment: "Gen 3", Parents: []envelopes.ID{gen2aid, gen2bid}}
		gen3id := gen3.ID()
		if err := repo.WriteTransaction(ctx, gen3); err != nil {
			t.Error(err)
			return
		}

		expected := make(map[envelopes.ID]struct{}, 3)
		expected[gen2aid] = struct{}{}
		expected[gen2bid] = struct{}{}
		expected[gen3id] = struct{}{}

		subject := Walker{Loader: repo, MaxDepth: 1}

		action := func(ctx context.Context, id envelopes.ID, transaction envelopes.Transaction) error {
			if _, ok := expected[id]; ok {
				delete(expected, id)
			} else {
				t.Errorf("unexpected transaction encountered:\n\tID:   %s\n\tDesc: %s", id, transaction.Comment)
			}
			return nil
		}

		if err := subject.Walk(ctx, action, gen3id); err != nil {
			t.Error(err)
			return
		}

		for k, _ := range expected {
			var missed envelopes.Transaction
			if err := repo.LoadTransaction(ctx, k, &missed); err != nil {
				t.Errorf("missed expected transaction:\n\tID:   %s\n\tDesc: failed to load", k)
				continue
			}

			t.Errorf("missed expected transaction:\n\tID:   %s\n\tDesc: %s", k, missed.Comment)
		}
	}
}
