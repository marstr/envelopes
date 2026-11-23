package persist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/marstr/envelopes"
)

type loggingRepository struct {
	*MockRepository
	logger       *slog.Logger
	failureStage string
	failure      error
}

func (repo *loggingRepository) DiagnosticLogger() *slog.Logger { return repo.logger }

func (repo *loggingRepository) WriteTransaction(ctx context.Context, transaction envelopes.Transaction) error {
	if repo.failureStage == "write_transaction" {
		return repo.failure
	}
	return repo.MockRepository.WriteTransaction(ctx, transaction)
}

func (repo *loggingRepository) WriteBranch(ctx context.Context, name string, id envelopes.ID) error {
	if repo.failureStage == "write_branch" {
		return repo.failure
	}
	return repo.MockRepository.WriteBranch(ctx, name, id)
}

func (repo *loggingRepository) SetCurrent(ctx context.Context, ref RefSpec) error {
	if repo.failureStage == "set_current" {
		return repo.failure
	}
	return repo.MockRepository.SetCurrent(ctx, ref)
}

func diagnosticRecords(t *testing.T, logs *bytes.Buffer) []map[string]interface{} {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
	var records []map[string]interface{}
	for {
		var record map[string]interface{}
		err := decoder.Decode(&record)
		if err == io.EOF {
			return records
		}
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
}

func findDiagnostic(t *testing.T, records []map[string]interface{}, operation, stage string) map[string]interface{} {
	t.Helper()
	for _, record := range records {
		if record["operation"] == operation && record["stage"] == stage {
			return record
		}
	}
	t.Fatalf("missing %s/%s diagnostic in %v", operation, stage, records)
	return nil
}

func TestCommitDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name    string
		branch  bool
		failure string
	}{
		{name: "detached"},
		{name: "branch", branch: true},
		{name: "transaction failure", failure: "write_transaction"},
		{name: "branch failure", branch: true, failure: "write_branch"},
		{name: "current failure", failure: "set_current"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			var logs bytes.Buffer
			failure := errors.New("write failed")
			mock := NewMockRepository(1, 10)
			if tc.branch {
				if err := mock.WriteBranch(ctx, DefaultBranch, envelopes.ID{}); err != nil {
					t.Fatal(err)
				}
				if err := mock.SetCurrent(ctx, RefSpec(DefaultBranch)); err != nil {
					t.Fatal(err)
				}
			}
			repo := &loggingRepository{
				MockRepository: mock,
				logger:         slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})).With("command", "commit"),
				failureStage:   tc.failure,
				failure:        failure,
			}
			const comment = "private-commit-comment-sentinel"
			err := Commit(ctx, repo, envelopes.Transaction{Comment: comment})
			records := diagnosticRecords(t, &logs)
			if tc.failure != "" {
				if err != failure {
					t.Fatalf("commit error changed: %v", err)
				}
				record := findDiagnostic(t, records, "commit", tc.failure)
				if record["error"] != failure.Error() {
					t.Fatalf("failure diagnostic: %v", record)
				}
				for _, record := range records {
					if record["operation"] == "commit" && record["stage"] == "complete" {
						t.Fatal("failed commit claimed completion")
					}
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				findDiagnostic(t, records, "commit", "complete")
			}
			if tc.failure != "write_transaction" {
				record := findDiagnostic(t, records, "commit", "transaction_written")
				if record["object_id"] == nil || record["command"] != "commit" {
					t.Fatalf("missing commit context: %v", record)
				}
				record = findDiagnostic(t, records, "commit", "reference_mode")
				mode := "detached"
				if tc.branch {
					mode = "branch"
				}
				if record["mode"] != mode {
					t.Fatalf("reference mode: %v", record)
				}
			}
			for _, record := range records {
				if record["level"] != "DEBUG" {
					t.Fatalf("unexpected log level: %v", record)
				}
			}
			if strings.Contains(logs.String(), comment) {
				t.Fatal("logged transaction comment")
			}
		})
	}
}

func TestBareCloneDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name       string
		failure    bool
		sourceOnly bool
	}{
		{name: "success"},
		{name: "partial failure", failure: true},
		{name: "source logger", sourceOnly: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			var sourceLogs, destinationLogs bytes.Buffer
			src := &loggingRepository{
				MockRepository: NewMockRepository(1, 10),
				logger:         slog.New(slog.NewJSONHandler(&sourceLogs, &slog.HandlerOptions{Level: slog.LevelDebug})),
			}
			parent := envelopes.Transaction{Comment: "private-clone-parent-sentinel"}
			child := envelopes.Transaction{Comment: "private-clone-child-sentinel", Parents: []envelopes.ID{parent.ID()}}
			for _, transaction := range []envelopes.Transaction{parent, child} {
				if err := src.WriteTransaction(ctx, transaction); err != nil {
					t.Fatal(err)
				}
			}
			if err := src.WriteBranch(ctx, DefaultBranch, child.ID()); err != nil {
				t.Fatal(err)
			}
			if err := src.WriteTag(ctx, "tag", Tag{ID: parent.ID(), Comment: "private-tag-sentinel"}); err != nil {
				t.Fatal(err)
			}
			dest := &loggingRepository{
				MockRepository: NewMockRepository(1, 10),
				logger:         slog.New(slog.NewJSONHandler(&destinationLogs, &slog.HandlerOptions{Level: slog.LevelDebug})),
			}
			if tc.sourceOnly {
				dest.logger = nil
			}
			if tc.failure {
				dest.failureStage = "write_transaction"
				dest.failure = errors.New("transaction write failed")
			}
			err := BareClone(ctx, src, dest)
			logs := &destinationLogs
			if tc.sourceOnly {
				logs = &sourceLogs
			} else if sourceLogs.Len() != 0 {
				t.Fatal("clone summary was duplicated in source logger")
			}
			stage := "complete"
			count := float64(2)
			if tc.failure {
				if err != dest.failure {
					t.Fatalf("clone error changed: %v", err)
				}
				stage, count = "write_transaction", 0
			} else if err != nil {
				t.Fatal(err)
			}
			record := findDiagnostic(t, diagnosticRecords(t, logs), "clone", stage)
			if record["branches_written"] != float64(1) || record["tags_written"] != float64(1) ||
				record["transactions_written"] != count {
				t.Fatalf("partial-progress counts: %v", record)
			}
			for _, private := range []string{parent.Comment, child.Comment, "private-tag-sentinel"} {
				if strings.Contains(logs.String(), private) {
					t.Fatal("logged cloned financial metadata")
				}
			}
		})
	}
}

func TestOrchestrationDiagnosticsDisabled(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	ctx := context.Background()
	for _, repo := range []RepositoryReaderWriter{
		NewMockRepository(1, 10),
		&loggingRepository{MockRepository: NewMockRepository(1, 10)},
	} {
		if err := Commit(ctx, repo, envelopes.Transaction{}); err != nil {
			t.Fatal(err)
		}
		if err := BareClone(ctx, repo, NewMockRepository(1, 10)); err != nil {
			t.Fatal(err)
		}
	}
	if logs.Len() != 0 {
		t.Fatal("orchestration diagnostics used the default logger")
	}
}

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

func TestBareClone_WithTags(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const transactionCount = 3
	const branchCount = 2
	const tagCount = 2
	src := NewMockRepository(branchCount+tagCount, transactionCount)

	a := envelopes.Transaction{Comment: "First"}
	if err := src.WriteTransaction(ctx, a); err != nil {
		t.Error(err)
		return
	}

	b := envelopes.Transaction{Comment: "Second", Parents: []envelopes.ID{a.ID()}}
	if err := src.WriteTransaction(ctx, b); err != nil {
		t.Error(err)
		return
	}

	c := envelopes.Transaction{Comment: "Third", Parents: []envelopes.ID{b.ID()}}
	if err := src.WriteTransaction(ctx, c); err != nil {
		t.Error(err)
		return
	}

	if err := src.WriteBranch(ctx, DefaultBranch, c.ID()); err != nil {
		t.Error(err)
		return
	}

	if err := src.WriteBranch(ctx, "develop", b.ID()); err != nil {
		t.Error(err)
		return
	}

	if err := src.WriteTag(ctx, "v1.0.0", Tag{ID: a.ID(), Comment: "First release"}); err != nil {
		t.Error(err)
		return
	}

	if err := src.WriteTag(ctx, "v2.0.0", Tag{ID: c.ID(), Comment: "Second release"}); err != nil {
		t.Error(err)
		return
	}

	dest := NewMockRepository(branchCount+tagCount, transactionCount)

	if err := BareClone(ctx, src, dest); err != nil {
		t.Error(err)
		return
	}

	// Verify branches were cloned
	readBranches, err := dest.ListBranches(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	branchesFound := make(map[string]bool)
	for branch := range readBranches {
		branchesFound[branch] = true
		id, err := dest.ReadBranch(ctx, branch)
		if err != nil {
			t.Error(err)
			return
		}
		expectedID, err := src.ReadBranch(ctx, branch)
		if err != nil {
			t.Error(err)
			return
		}
		if !id.Equal(expectedID) {
			t.Logf("Branch %q: got %q, want %q", branch, id, expectedID)
			t.Fail()
		}
	}

	if !branchesFound[DefaultBranch] || !branchesFound["develop"] {
		t.Logf("Missing branches: %v", branchesFound)
		t.Fail()
	}

	// Verify tags were cloned
	readTags, err := dest.ListTags(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	tagsFound := make(map[string]bool)
	for tag := range readTags {
		tagsFound[tag] = true
		gotTag, err := dest.ReadTag(ctx, tag)
		if err != nil {
			t.Error(err)
			return
		}
		expectedTag, err := src.ReadTag(ctx, tag)
		if err != nil {
			t.Error(err)
			return
		}
		if !gotTag.ID.Equal(expectedTag.ID) {
			t.Logf("Tag %q ID: got %q, want %q", tag, gotTag.ID, expectedTag.ID)
			t.Fail()
		}
		if gotTag.Comment != expectedTag.Comment {
			t.Logf("Tag %q Comment: got %q, want %q", tag, gotTag.Comment, expectedTag.Comment)
			t.Fail()
		}
	}

	if !tagsFound["v1.0.0"] || !tagsFound["v2.0.0"] {
		t.Logf("Missing tags: %v", tagsFound)
		t.Fail()
	}

	// Verify transactions were cloned
	var loaded envelopes.Transaction
	if err := dest.LoadTransaction(ctx, a.ID(), &loaded); err != nil {
		t.Error(err)
		return
	}
	if loaded.Comment != "First" {
		t.Logf("Transaction a not cloned correctly")
		t.Fail()
	}

	if err := dest.LoadTransaction(ctx, c.ID(), &loaded); err != nil {
		t.Error(err)
		return
	}
	if loaded.Comment != "Third" {
		t.Logf("Transaction c not cloned correctly")
		t.Fail()
	}
}
