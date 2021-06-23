package persist

import (
	"context"
	"github.com/marstr/envelopes"
	"testing"
)

func TestBareClone(t *testing.T) {
	//ctx, cancel := context.WithTimeout(context.Background(), 90 * time.Second)
	// defer cancel()
	ctx := context.Background()

	t.Run("linear", testBareCloneLinear(ctx))
}

func testBareCloneLinear(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		const transactionCount = 4
		const branchCount = 1
		src := NewMockRepository(branchCount, transactionCount)
		expected := make(map[envelopes.ID]envelopes.Transaction)

		a := envelopes.Transaction{Comment: "Deepest"}
		if err := src.Write(ctx, a); err != nil {
			t.Error(err)
		}

		b := envelopes.Transaction{Comment: "Deeper", Parents: []envelopes.ID{a.ID()}}
		if err := src.Write(ctx, b); err != nil {
			t.Error(err)
		}
		expected[b.Parents[0]] = a

		c := envelopes.Transaction{Comment: "Deep", Parents: []envelopes.ID{b.ID()}}
		if err := src.Write(ctx, c); err != nil {
			t.Error(err)
		}
		expected[c.Parents[0]] = b

		d := envelopes.Transaction{Comment: "Shallow", Parents: []envelopes.ID{c.ID()}}
		if err := src.Write(ctx, d); err != nil {
			t.Error(err)
		}
		expected[d.Parents[0]] = c
		dId := d.ID()
		expected[dId] = d
		if err := src.WriteBranch(ctx, DefaultBranch, dId); err != nil {
			t.Error(err)
			return
		}

		dest := NewMockRepository(branchCount, transactionCount)

		if err := BareClone(ctx, src, dest); err != nil {
			t.Error(err)
			return
		}

		destBranchList, err := dest.ListBranches(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		for foundBranch := range destBranchList {
			t.Logf("Found a branch! %q", foundBranch)
		}

		for id, want := range expected {
			var got envelopes.Transaction
			err = dest.Load(ctx, id, &got)
			if err != nil {
				t.Error(err)
				continue
			}

			if !got.Equal(want) {
				t.Errorf("Transaction %s did not match expected", id)
			}
		}
	}
}
