package traverse

import (
	"context"
	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
	"testing"
)

func TestWalker_Walk(t *testing.T) {
	ctx := context.Background()
	t.Run("single line", chain(ctx))
	t.Run("basic fork", fork(ctx))
}

func chain(ctx context.Context) func(t *testing.T) {
	cache := persist.NewCache(2)
	a := envelopes.Transaction{
		Comment: "First!",
	}
	aid := a.ID()
	err := cache.Write(ctx, a)
	if err != nil {
		panic(err)
	}

	b := envelopes.Transaction{
		Comment: "Second!",
		Parent: []envelopes.ID{
			aid,
		},
	}
	bid := b.ID()
	err = cache.Write(ctx, b)
	if err != nil {
		panic(err)
	}

	return func (t *testing.T) {
		expected := map[envelopes.ID]struct{}{
			aid: {},
			bid: {},
		}

		texasRanger := Walker{
			Loader: cache,
		}

		err := texasRanger.Walk(ctx, func(ctx context.Context, transaction envelopes.Transaction) error {
			currentId := transaction.ID()
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
	cache := persist.NewCache(3)
	a := envelopes.Transaction{
		Comment: "First!",
	}
	aid := a.ID()
	err := cache.Write(ctx, a)
	if err != nil {
		panic(err)
	}

	b := envelopes.Transaction{
		Comment: "Second!",
		Parent: []envelopes.ID{
			aid,
		},
	}
	bid := b.ID()
	err = cache.Write(ctx, b)
	if err != nil {
		panic(err)
	}

	c := envelopes.Transaction{
		Comment: "Third!",
		Parent: []envelopes.ID{
			aid,
		},
	}
	cid := c.ID() // Eastern Iowa Airport shout-out
	err = cache.Write(ctx, c)
	if err != nil {
		panic(err)
	}

	return func (t *testing.T) {
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

		err := texasRanger.Walk(ctx, func(ctx context.Context, transaction envelopes.Transaction) error {
			currentId := transaction.ID()
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

