package filesystem_test

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
	"github.com/marstr/envelopes/persist/filesystem"
)

func TestOpenRepositoryLayout1(t *testing.T) {
	var ctx context.Context

	if deadline, ok := t.Deadline(); ok {
		const deleteFilesTime = -3 * time.Second
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline.Add(deleteFilesTime))
		defer cancel()
	} else {
		ctx = context.Background()
	}

	repo, err := filesystem.OpenRepository(ctx, "./testdata/test5/.baronial")
	if err != nil {
		t.Error(err)
	}

	encounteredLayout := repo.FileSystem.ObjectLayout
	expectedLayout := uint(1)

	if encounteredLayout != expectedLayout {
		t.Errorf("wrong layout\n\tgot: %v\n\twant: %v", encounteredLayout, expectedLayout)
	}

	current, err := repo.Current(ctx)
	if err != nil {
		t.Error(err)
	}

	headId, err := persist.Resolve(ctx, repo, current)
	if err != nil {
		t.Error(err)
	}

	var head envelopes.Transaction
	err = repo.LoadTransaction(ctx, headId, &head)
	if err != nil {
		t.Error(err)
	}

	encounteredStateId := head.State.ID().String()
	expectedStateId := "960a403e64cca0c8022c8d72e96905991b74f533"
	if encounteredStateId != expectedStateId {
		t.Errorf("wrong transaction:\n\texpected state: %q\n\tgot state %q", expectedStateId, encounteredStateId)
	}
}

func TestCreateRepositoryLayout1(t *testing.T) {
	var ctx context.Context

	if deadline, ok := t.Deadline(); ok {
		const deleteFilesTime = -3 * time.Second
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline.Add(deleteFilesTime))
		defer cancel()
	} else {
		ctx = context.Background()
	}

	testDir, err := os.MkdirTemp("", "envelopes")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(testDir)

	repo, err := filesystem.OpenRepository(ctx, testDir, filesystem.RepositoryObjectLoc(1))
	if err != nil {
		t.Error(err)
	}

	exampleTransaction := envelopes.Transaction{
		State: &envelopes.State{
			Budget: &envelopes.Budget{
				Balance: envelopes.Balance{"": big.NewRat(314, 100)},
			},
		},
	}

	err = repo.WriteTransaction(ctx, exampleTransaction)
	if err != nil {
		t.Error(err)
	}

	id := exampleTransaction.ID().String()

	handle, err := os.Open(filepath.Join(testDir, filesystem.ObjectsDir, id[:2], id[2:]+".json"))
	if err != nil {
		t.Error(err)
	}
	defer handle.Close()
}
