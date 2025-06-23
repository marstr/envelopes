package persist

import (
	"context"
	"testing"
	"time"

	"github.com/marstr/envelopes"
)

func TestBareClone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	t.Run("linear", testBareCloneLinear(ctx))
	t.Run("diamond", testBareCloneDiamond(ctx))
	t.Run("fork", testBareCloneFork(ctx))
}

func testBareCloneLinear(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		const transactionCount = 4
		const branchCount = 1
		src := NewMockRepository(branchCount, transactionCount)
		expected := make(map[envelopes.ID]envelopes.Transaction)

		a := envelopes.Transaction{Comment: "Deepest"}
		if err := src.WriteTransaction(ctx, a); err != nil {
			t.Error(err)
		}

		b := envelopes.Transaction{Comment: "Deeper", Parents: []envelopes.ID{a.ID()}}
		if err := src.WriteTransaction(ctx, b); err != nil {
			t.Error(err)
		}
		expected[b.Parents[0]] = a

		c := envelopes.Transaction{Comment: "Deep", Parents: []envelopes.ID{b.ID()}}
		if err := src.WriteTransaction(ctx, c); err != nil {
			t.Error(err)
		}
		expected[c.Parents[0]] = b

		d := envelopes.Transaction{Comment: "Shallow", Parents: []envelopes.ID{c.ID()}}
		if err := src.WriteTransaction(ctx, d); err != nil {
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

		faithfulClone, err := bareRepositoriesEqual(ctx, src, dest)
		if err != nil {
			t.Error(err)
			return
		}
		if !faithfulClone {
			t.Error("The subject repositories didn't match after a BareClone operation")
		}
	}
}

func testBareCloneDiamond(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		const transactionCount = 4
		const branchCount = 1
		src := NewMockRepository(branchCount, transactionCount)

		a := envelopes.Transaction{Comment: "Deepest"}
		if err := src.WriteTransaction(ctx, a); err != nil {
			t.Error(err)
		}
		aId := a.ID()

		b := envelopes.Transaction{Comment: "Deeper", Parents: []envelopes.ID{aId}}
		if err := src.WriteTransaction(ctx, b); err != nil {
			t.Error(err)
		}
		bId := b.ID()

		c := envelopes.Transaction{Comment: "Deep", Parents: []envelopes.ID{aId}}
		if err := src.WriteTransaction(ctx, c); err != nil {
			t.Error(err)
		}
		cId := c.ID()

		d := envelopes.Transaction{Comment: "Shallow", Parents: []envelopes.ID{cId, bId}}
		if err := src.WriteTransaction(ctx, d); err != nil {
			t.Error(err)
		}
		dId := d.ID()
		if err := src.WriteBranch(ctx, DefaultBranch, dId); err != nil {
			t.Error(err)
			return
		}

		dest := NewMockRepository(branchCount, transactionCount)

		if err := BareClone(ctx, src, dest); err != nil {
			t.Error(err)
			return
		}

		faithfulClone, err := bareRepositoriesEqual(ctx, src, dest)
		if err != nil {
			t.Error(err)
			return
		}
		if !faithfulClone {
			t.Error("The subject repositories didn't match after a BareClone operation")
		}
	}
}

func testBareCloneFork(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		const transactionCount = 3
		const branchCount = 1
		src := NewMockRepository(branchCount, transactionCount)

		b := envelopes.Transaction{Comment: "Deeper", Parents: []envelopes.ID{}}
		if err := src.WriteTransaction(ctx, b); err != nil {
			t.Error(err)
		}
		bId := b.ID()

		c := envelopes.Transaction{Comment: "Deep", Parents: []envelopes.ID{}}
		if err := src.WriteTransaction(ctx, c); err != nil {
			t.Error(err)
		}
		cId := c.ID()

		d := envelopes.Transaction{Comment: "Shallow", Parents: []envelopes.ID{cId, bId}}
		if err := src.WriteTransaction(ctx, d); err != nil {
			t.Error(err)
		}
		dId := d.ID()
		if err := src.WriteBranch(ctx, DefaultBranch, dId); err != nil {
			t.Error(err)
			return
		}

		dest := NewMockRepository(branchCount, transactionCount)

		if err := BareClone(ctx, src, dest); err != nil {
			t.Error(err)
			return
		}

		faithfulClone, err := bareRepositoriesEqual(ctx, src, dest)
		if err != nil {
			t.Error(err)
			return
		}
		if !faithfulClone {
			t.Error("The subject repositories didn't match after a BareClone operation")
		}
	}
}

func bareRepositoriesEqual(ctx context.Context, left, right BareRepositoryReader) (bool, error) {
	if left == right {
		return true, nil
	}

	// Ensure branch lists are the same, preserve a list of them to act as heads for a full traversal.
	expectedBranchesRaw, err := left.ListBranches(ctx)
	if err != nil {
		return false, err
	}
	expectedBranches := make(map[string]struct{})
	for branchName := range expectedBranchesRaw {
		expectedBranches[branchName] = struct{}{}
	}

	foundBranches := make([]envelopes.ID, 0, len(expectedBranches))
	destBranchList, err := right.ListBranches(ctx)
	if err != nil {
		return false, err
	}
	for foundBranch := range destBranchList {
		if _, ok := expectedBranches[foundBranch]; !ok {
			return false, nil
		}

		id, err := left.ReadBranch(ctx, foundBranch)
		if err != nil {
			return false, err
		}
		foundBranches = append(foundBranches, id)
		delete(expectedBranches, foundBranch)
	}
	if len(expectedBranches) > 0 {
		return false, nil
	}

	// Walk through all transactions in the left repository, and make sure they're all present in the right repository.
	walker := Walker{Loader: left}
	var current envelopes.Transaction
	err = walker.Walk(ctx, func(ctx context.Context, id envelopes.ID, transaction envelopes.Transaction) error {
		err = right.LoadTransaction(ctx, id, &current)
		if err != nil {
			return err
		}
		return nil
	}, foundBranches...)
	if err != nil {
		return false, err
	}

	// Walk through all transactions in the right repository, and make sure they're all present in the left repository.
	walker.Loader = right
	err = walker.Walk(ctx, func(ctx context.Context, id envelopes.ID, transaction envelopes.Transaction) error {
		err = left.LoadTransaction(ctx, id, &current)
		if err != nil {
			return err
		}
		return nil
	}, foundBranches...)
	if err != nil {
		return false, err
	}

	return true, nil
}

func TestCommit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("firstHeadless", testFirstHeadlessCommit(ctx))
	t.Run("firstBranched", testFirstBranchedCommit(ctx))
	t.Run("secondHeadless", testSecondHeadlessCommit(ctx))
	t.Run("secondBranched", testSecondBranchedCommit(ctx))
}

func testFirstHeadlessCommit(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		src := NewMockRepository(1, 1)

		const comment = "First Transaction"
		transaction := envelopes.Transaction{Comment: comment}

		expected := envelopes.Transaction{Comment: comment, Parents: []envelopes.ID{}}

		err := Commit(ctx, src, transaction)
		if err != nil {
			t.Error(err)
			return
		}

		// Make sure it actually wrote the transaction
		var actualTransaction envelopes.Transaction
		err = src.LoadTransaction(ctx, expected.ID(), &actualTransaction)
		if err != nil {
			t.Error(err)
			return
		}
		if !expected.Equal(actualTransaction) {
			t.Error("didn't get the expected transaction")
		}

		// Make sure Current was set to the transaction's ID
		current, err := src.Current(ctx)
		if err != nil {
			t.Error(err)
			return
		}

		if string(current) != expected.ID().String() {
			t.Errorf("Current wasn't set correctly.\ngot:  \"%s\"\nwant: \"%s\"", string(current), expected.ID().String())
		}

		// Make sure it doesn't create any branches
		readBranches, err := src.ListBranches(ctx)
		if err != nil {
			t.Error(err)
		}

		actualBranches := []string{}
		for branch := range readBranches {
			actualBranches = append(actualBranches, branch)
		}

		if len(actualBranches) > 0 {
			t.Errorf("There should have been no branches but instead it had the following %d: %v", len(actualBranches), actualBranches)
		}
	}
}

func testFirstBranchedCommit(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		const defaultBranch = "kirk"
		src := NewMockRepository(1, 1)
		err := src.WriteBranch(ctx, defaultBranch, envelopes.ID{})
		if err != nil {
			t.Error(err)
			return
		}
		src.SetCurrent(ctx, defaultBranch)

		const comment = "First Transaction"
		transaction := envelopes.Transaction{Comment: comment}

		expected := envelopes.Transaction{Comment: comment, Parents: []envelopes.ID{}}
		expectedId := expected.ID()

		err = Commit(ctx, src, transaction)
		if err != nil {
			t.Error(err)
			return
		}

		// Make sure it actually wrote the transaction
		var actualTransaction envelopes.Transaction
		err = src.LoadTransaction(ctx, expectedId, &actualTransaction)
		if err != nil {
			t.Error(err)
			return
		}
		if !expected.Equal(actualTransaction) {
			t.Error("didn't get the expected transaction")
		}

		// Make sure Current is still set to the default branch.
		current, err := src.Current(ctx)
		if err != nil {
			t.Error(err)
			return
		}

		if string(current) != defaultBranch {
			t.Errorf("Current wasn't set correctly.\ngot:  \"%s\"\nwant: \"%s\"", string(current), defaultBranch)
		}

		// Make sure the branch got updated

		actualId, err := src.ReadBranch(ctx, defaultBranch)
		if err != nil {
			t.Error(err)
		}
		if !actualId.Equal(expectedId) {
			t.Errorf("the branch didn't get updated as expected.\ngot:  %q\nwant: %q", actualId.String(), expectedId.String())
		}

		// Make sure it doesn't change the branches in the repository
		readBranches, err := src.ListBranches(ctx)
		if err != nil {
			t.Error(err)
		}

		actualBranches := []string{}
		for branch := range readBranches {
			actualBranches = append(actualBranches, branch)
		}

		if len(actualBranches) != 1 {
			t.Errorf("There should have been exactly one branch but instead it had the following %d: %v", len(actualBranches), actualBranches)
			return
		}

		if actualBranches[0] != defaultBranch {
			t.Errorf("It renamed the branch to %q when it should have been %q", actualBranches[0], defaultBranch)
		}
	}
}

func testSecondHeadlessCommit(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		// Setup
		src := NewMockRepository(2, 2)
		root := envelopes.Transaction{Comment: "First Transaction", Parents: []envelopes.ID{}}
		src.WriteTransaction(ctx, root)
		err := src.SetCurrent(ctx, RefSpec(root.ID().String()))
		if err != nil {
			t.Error(err)
			return
		}

		const addComment = "Second Transaction"
		added := envelopes.Transaction{Comment: addComment}

		err = Commit(ctx, src, added)
		if err != nil {
			t.Error(err)
			return
		}

		// Make sure it actually wrote the transaction
		expected := envelopes.Transaction{Comment: addComment, Parents: []envelopes.ID{root.ID()}}
		expectedId := expected.ID()
		var actualTransaction envelopes.Transaction
		err = src.LoadTransaction(ctx, expectedId, &actualTransaction)
		if err != nil {
			t.Error(err)
			return
		}
		if !expected.Equal(actualTransaction) {
			t.Error("didn't get the expected transaction")
		}

		// Make sure Current was set to the transaction's ID
		current, err := src.Current(ctx)
		if err != nil {
			t.Error(err)
			return
		}

		if string(current) != expectedId.String() {
			t.Errorf("Current wasn't set correctly.\ngot:  \"%s\"\nwant: \"%s\"", string(current), expectedId.String())
		}
	}
}

func testSecondBranchedCommit(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		// Setup
		const defaultBranch = "picard"
		src := NewMockRepository(2, 2)
		root := envelopes.Transaction{Comment: "First Transaction", Parents: []envelopes.ID{}}
		rootId := root.ID()
		src.WriteTransaction(ctx, root)
		src.WriteBranch(ctx, defaultBranch, rootId)
		err := src.SetCurrent(ctx, defaultBranch)
		if err != nil {
			t.Error(err)
			return
		}

		const addComment = "Second Transaction"
		added := envelopes.Transaction{Comment: addComment}

		// Execute
		err = Commit(ctx, src, added)
		if err != nil {
			t.Error(err)
			return
		}

		// Make sure it actually wrote the transaction
		expected := envelopes.Transaction{Comment: addComment, Parents: []envelopes.ID{root.ID()}}
		expectedId := expected.ID()
		var actualTransaction envelopes.Transaction
		err = src.LoadTransaction(ctx, expectedId, &actualTransaction)
		if err != nil {
			t.Error(err)
			return
		}
		if !expected.Equal(actualTransaction) {
			t.Error("didn't get the expected transaction")
		}

		// Make sure Current is still set to the default branch.
		current, err := src.Current(ctx)
		if err != nil {
			t.Error(err)
			return
		}

		if string(current) != defaultBranch {
			t.Errorf("Current wasn't set correctly.\ngot:  \"%s\"\nwant: \"%s\"", string(current), defaultBranch)
		}

		// Make sure the branch got updated

		actualId, err := src.ReadBranch(ctx, defaultBranch)
		if err != nil {
			t.Error(err)
		}
		if !actualId.Equal(expectedId) {
			t.Errorf("the branch didn't get updated as expected.\ngot:  %q\nwant: %q", actualId.String(), expectedId.String())
		}

		// Make sure it doesn't change the branches in the repository
		readBranches, err := src.ListBranches(ctx)
		if err != nil {
			t.Error(err)
		}

		actualBranches := []string{}
		for branch := range readBranches {
			actualBranches = append(actualBranches, branch)
		}

		if len(actualBranches) != 1 {
			t.Errorf("There should have been exactly one branch but instead it had the following %d: %v", len(actualBranches), actualBranches)
			return
		}

		if actualBranches[0] != defaultBranch {
			t.Errorf("It renamed the branch to %q when it should have been %q", actualBranches[0], defaultBranch)
		}
	}
}
