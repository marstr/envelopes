package filesystem_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
	"github.com/marstr/envelopes/persist/filesystem"
	persistJSON "github.com/marstr/envelopes/persist/json"
)

func TestRepository_LoggerPropagation(t *testing.T) {
	ctx := context.Background()
	for version := uint(1); version <= 3; version++ {
		for _, cached := range []bool{false, true} {
			t.Run(fmt.Sprintf("v%d/cache=%v", version, cached), func(t *testing.T) {
				root := t.TempDir()
				fs := filesystem.FileSystem{Root: root}
				var writer persist.Writer
				var err error
				switch version {
				case 1:
					writer, err = persistJSON.NewWriterV1(fs)
				case 2:
					writer, err = persistJSON.NewWriterV2(fs)
				case 3:
					writer, err = persistJSON.NewWriterV3(fs)
				}
				if err != nil {
					t.Fatal(err)
				}
				transaction := envelopes.Transaction{
					State: &envelopes.State{
						Budget: &envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(1, 1)}},
					},
					Comment:  "private-transaction-comment-sentinel",
					Merchant: "private-merchant-sentinel",
					Committer: envelopes.User{
						FullName: "private-name-sentinel",
						Email:    "private-email-sentinel",
					},
				}
				if err := writer.WriteTransaction(ctx, transaction); err != nil {
					t.Fatal(err)
				}
				config, err := json.Marshal(filesystem.RepositoryConfig{
					Objects: filesystem.RepositoryConfigEntry{Format: filesystem.FormatJson, Version: version},
				})
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, filesystem.ConfigFilename), config, 0600); err != nil {
					t.Fatal(err)
				}

				var logs bytes.Buffer
				logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})).
					With("command", "test")
				var repo *filesystem.Repository
				if cached {
					repo, err = filesystem.OpenRepositoryWithCache(ctx, root, 10, filesystem.RepositoryLogger(logger))
				} else {
					repo, err = filesystem.OpenRepository(ctx, root, filesystem.RepositoryLogger(logger))
				}
				if err != nil {
					t.Fatal(err)
				}
				record := operationRecord(t, logRecords(t, &logs), "open_repository", "configuration")
				if record["version"] != float64(version) || record["cache_enabled"] != cached {
					t.Fatalf("configuration diagnostic: %v", record)
				}
				logs.Reset()
				var loaded envelopes.Transaction
				if err := repo.LoadTransaction(ctx, transaction.ID(), &loaded); err != nil {
					t.Fatal(err)
				}
				record = operationRecord(t, logRecords(t, &logs), "fetch", "read")
				if record["command"] != "test" {
					t.Fatalf("caller attributes lost: %v", record)
				}
				if cached {
					logs.Reset()
					if err := repo.LoadTransaction(ctx, transaction.ID(), &loaded); err != nil {
						t.Fatal(err)
					}
					if logs.Len() != 0 {
						t.Fatal("cache hit unexpectedly read from disk")
					}
				}
				logs.Reset()
				transaction.Comment += "-updated"
				if err := repo.WriteTransaction(ctx, transaction); err != nil {
					t.Fatal(err)
				}
				operationRecord(t, logRecords(t, &logs), "stash", "write")
				for _, private := range []string{transaction.Comment, transaction.Merchant,
					transaction.Committer.FullName, transaction.Committer.Email} {
					if strings.Contains(logs.String(), private) {
						t.Fatal("logged financial metadata")
					}
				}
			})
		}
	}
}

func TestRepository_OptionsReachBackingFilesystem(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	repo, err := filesystem.OpenRepository(ctx, root,
		filesystem.RepositoryObjectLoc(0), filesystem.RepositoryFileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	transaction := envelopes.Transaction{Comment: "layout regression"}
	if err := repo.WriteTransaction(ctx, transaction); err != nil {
		t.Fatal(err)
	}
	loc := filepath.Join(root, filesystem.ObjectsDir, transaction.ID().String()+".json")
	info, err := os.Stat(loc)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("object permissions: got %v, want 0600", info.Mode().Perm())
	}
	config, err := filesystem.LoadConfig(ctx, root)
	if err != nil || config.ObjectLocations != 0 {
		t.Fatalf("saved layout: %v, %v", config, err)
	}
	reopened, err := filesystem.OpenRepository(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	var loaded envelopes.Transaction
	if err := reopened.LoadTransaction(ctx, transaction.ID(), &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.ID() != transaction.ID() {
		t.Fatal("reopened repository used the wrong layout")
	}
	defaultRepo, err := filesystem.OpenRepository(ctx, t.TempDir())
	if err != nil || defaultRepo.ObjectLayout != 1 {
		t.Fatalf("global default configuration changed: %v, %v", defaultRepo, err)
	}
	if err := defaultRepo.WriteTransaction(ctx, transaction); err != nil {
		t.Fatal(err)
	}
	if _, err := filesystem.OpenRepository(ctx, defaultRepo.Root, filesystem.RepositoryObjectLoc(0)); err == nil {
		t.Fatal("changing an existing layout would strand its objects")
	}
	defaultRepo, err = filesystem.OpenRepository(ctx, defaultRepo.Root)
	if err != nil {
		t.Fatal(err)
	}
	if err := defaultRepo.LoadTransaction(ctx, transaction.ID(), &loaded); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_LoggerDisabledAndConfigWriteFailure(t *testing.T) {
	ctx := context.Background()
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	if _, err := filesystem.OpenRepository(ctx, t.TempDir(),
		filesystem.RepositoryLogger(logger), filesystem.RepositoryLogger(nil)); err != nil {
		t.Fatal(err)
	}
	if logs.Len() != 0 {
		t.Fatal("nil option did not disable logging")
	}

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, filesystem.ConfigFilename), 0700); err != nil {
		t.Fatal(err)
	}
	_, err := filesystem.OpenRepository(ctx, root, filesystem.RepositoryLogger(logger))
	if err == nil {
		t.Fatal("expected configuration write failure")
	}
	record := operationRecord(t, logRecords(t, &logs), "write_config", "write")
	if record["error"] != err.Error() {
		t.Fatalf("config write diagnostic: %v", record)
	}
	for _, record := range logRecords(t, &logs) {
		if record["operation"] == "open_repository" && record["stage"] == "complete" {
			t.Fatal("failed repository open claimed completion")
		}
	}
}

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
