// Copyright 2017 Martin Strobel
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package filesystem_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
	"github.com/marstr/envelopes/persist/filesystem"
)

func logRecords(t *testing.T, buf *bytes.Buffer) []map[string]interface{} {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	var records []map[string]interface{}
	for {
		var record map[string]interface{}
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func operationRecord(t *testing.T, records []map[string]interface{}, operation, stage string) map[string]interface{} {
	t.Helper()
	for _, record := range records {
		if record["operation"] == operation && record["stage"] == stage {
			return record
		}
	}
	t.Fatalf("missing %s/%s log in %v", operation, stage, records)
	return nil
}

func TestFileSystem_Logging(t *testing.T) {
	ctx := context.Background()
	var defaultLogs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&defaultLogs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo} {
		t.Run(level.String(), func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: level})).
				With("command", "test")
			fs := filesystem.FileSystem{Root: t.TempDir()}
			id := envelopes.ID{1}
			const payload = "private-payload-sentinel"
			if err := fs.Stash(ctx, id, []byte(payload)); err != nil {
				t.Fatal(err)
			}
			if defaultLogs.Len() != 0 {
				t.Fatal("nil filesystem logger used the default logger")
			}

			fs.Logger = logger
			raw, err := fs.Fetch(ctx, id)
			if err != nil || string(raw) != payload {
				t.Fatalf("fetch: %q, %v", raw, err)
			}
			if level == slog.LevelInfo {
				if logs.Len() != 0 {
					t.Fatal("Debug diagnostics were not filtered")
				}
				return
			}
			record := operationRecord(t, logRecords(t, &logs), "fetch", "read")
			if record["command"] != "test" || record["object_id"] != id.String() ||
				record["path"] != filepath.Join(fs.Root, filesystem.ObjectsDir, id.String()+".json") ||
				record["bytes"] != float64(len(payload)) {
				t.Fatalf("missing diagnostic context: %v", record)
			}
			if strings.Contains(logs.String(), payload) {
				t.Fatal("logged object payload")
			}

			logs.Reset()
			_, err = fs.Fetch(ctx, envelopes.ID{2})
			if !os.IsNotExist(err) {
				t.Fatalf("missing object error changed: %v", err)
			}
			record = operationRecord(t, logRecords(t, &logs), "fetch", "read")
			if record["level"] != "DEBUG" || record["error"] != err.Error() {
				t.Fatalf("missing object diagnostic: %v", record)
			}
		})
	}
}

type failingStream struct {
	readErr  error
	closeErr error
	closed   bool
}

func (stream *failingStream) Read([]byte) (int, error) { return 0, stream.readErr }
func (stream *failingStream) Close() error {
	stream.closed = true
	return stream.closeErr
}

func TestFileSystem_StreamLogging(t *testing.T) {
	ctx := context.Background()
	var logs bytes.Buffer
	fs := filesystem.FileSystem{
		Root:   t.TempDir(),
		Logger: slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	readErr := errors.New("source read failed")
	closeErr := errors.New("source close failed")
	stream := &failingStream{readErr: readErr, closeErr: closeErr}
	err := fs.StashReadCloser(ctx, envelopes.ID{3}, stream)
	if err != readErr || !stream.closed {
		t.Fatalf("stream error/ownership changed: %v, closed=%v", err, stream.closed)
	}
	records := logRecords(t, &logs)
	copyRecord := operationRecord(t, records, "stash_stream", "copy")
	if copyRecord["error"] != readErr.Error() || copyRecord["bytes"] != float64(0) {
		t.Fatalf("copy diagnostic: %v", copyRecord)
	}
	closeRecord := operationRecord(t, records, "stash_stream_source", "close")
	if closeRecord["level"] != "WARN" || closeRecord["error"] != closeErr.Error() {
		t.Fatalf("ignored close diagnostic: %v", closeRecord)
	}

	logs.Reset()
	const contents = "private-attachment-sentinel"
	if err := fs.StashReadCloser(ctx, envelopes.ID{4}, io.NopCloser(strings.NewReader(contents))); err != nil {
		t.Fatal(err)
	}
	handle, err := fs.FetchReadCloser(ctx, envelopes.ID{4})
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	record := operationRecord(t, logRecords(t, &logs), "fetch_stream", "open")
	if _, hasBytes := record["bytes"]; hasBytes {
		t.Fatal("stream opening claimed to have read bytes")
	}
	raw, err := io.ReadAll(handle)
	if err != nil || string(raw) != contents {
		t.Fatalf("stream fetch: %q, %v", raw, err)
	}
	if strings.Contains(logs.String(), contents) {
		t.Fatal("logged attachment content")
	}

	logs.Reset()
	blockedRoot := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blockedRoot, nil, 0600); err != nil {
		t.Fatal(err)
	}
	fs.Root = blockedRoot
	err = fs.Stash(ctx, envelopes.ID{5}, nil)
	if err == nil {
		t.Fatal("expected mkdir failure")
	}
	record = operationRecord(t, logRecords(t, &logs), "stash", "mkdir")
	if record["error"] != err.Error() {
		t.Fatalf("mkdir diagnostic: %v", record)
	}
}

func TestFileSystem_ReferenceLogging(t *testing.T) {
	ctx := context.Background()
	var logs bytes.Buffer
	fs := filesystem.FileSystem{
		Root:   t.TempDir(),
		Logger: slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	id := envelopes.ID{6}
	const comment = "private-tag-comment-sentinel"
	if err := fs.WriteTag(ctx, "tag", persist.Tag{ID: id, Comment: comment}); err != nil {
		t.Fatal(err)
	}
	tag, err := fs.ReadTag(ctx, "tag")
	if err != nil || tag.ID != id || tag.Comment != comment {
		t.Fatalf("tag: %v, %v", tag, err)
	}
	if _, err := fs.ReadBranch(ctx, "missing"); !os.IsNotExist(err) {
		t.Fatalf("branch miss: %v", err)
	}
	if _, err := fs.ReadTag(ctx, "missing"); !os.IsNotExist(err) {
		t.Fatalf("tag miss: %v", err)
	}
	for _, record := range logRecords(t, &logs) {
		if record["level"] != "DEBUG" {
			t.Fatalf("reference miss produced a warning/error: %v", record)
		}
	}
	if strings.Contains(logs.String(), comment) {
		t.Fatal("logged tag comment")
	}

	logs.Reset()
	tags, err := fs.ListTags(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for range tags {
	}
	record := operationRecord(t, logRecords(t, &logs), "list_tags", "enumerated")
	if record["count"] != float64(1) {
		t.Fatalf("listing count: %v", record)
	}

	logs.Reset()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	branches, err := fs.ListBranches(canceled)
	if err != nil {
		t.Fatal(err)
	}
	for range branches {
	}
	record = operationRecord(t, logRecords(t, &logs), "list_branches", "enumerated")
	if record["level"] != "DEBUG" || record["error"] != context.Canceled.Error() {
		t.Fatalf("listing cancellation: %v", record)
	}
}

func TestFileSystem_Current(t *testing.T) {
	var ctx context.Context

	if deadline, ok := t.Deadline(); ok {
		const deleteFilesTime = -3 * time.Second
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline.Add(deleteFilesTime))
		defer cancel()
	} else {
		ctx = context.Background()
	}

	testCases := []string{
		filepath.Join(".", "testdata", "test1"),
		filepath.Join(".", "testdata", "test2"),
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			repo, err := filesystem.OpenRepository(ctx, tc)
			if err != nil {
				t.Error(err)
				return
			}

			head, err := repo.Current(ctx)
			if err != nil {
				t.Error(err)
			}

			headId, err := persist.Resolve(ctx, repo, head)
			if err != nil {
				t.Error(err)
			}

			for i := range headId {
				if headId[i] != 0 {
					t.Logf("\ngot:  %X\nwant: %X", headId, envelopes.ID{})
					t.Fail()
					break
				}
			}
		})
	}
}

func TestFileSystem_RoundTrip_Current(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	testLocation := filepath.Join("testdata", "test", "roundtrip", "current")
	err := os.MkdirAll(testLocation, os.ModePerm)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	defer func() {
		err := os.RemoveAll(testLocation)
		if err != nil {
			t.Logf("failed to cleanup, directory %q still exists.", testLocation)
		}
	}()

	testCases := []envelopes.Transaction{
		{},
		{Comment: `"the time has come", the walrus said, "to speak of many things."`},
		{Amount: envelopes.Balance{"USD": big.NewRat(1729, 1)}},
	}

	repo, err := filesystem.OpenRepository(ctx, testLocation)
	if err != nil {
		t.Error(err)
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			err := repo.WriteTransaction(ctx, tc)
			if err != nil {
				t.Error(err)
				return
			}

			err = repo.FileSystem.SetCurrent(ctx, persist.RefSpec(tc.ID().String()))
			if err != nil {
				t.Error(err)
				return
			}

			refSpec, err := repo.Current(ctx)
			if err != nil {
				t.Error(err)
				return
			}

			got, err := persist.Resolve(ctx, repo, refSpec)
			if err != nil {
				t.Error(err)
				return
			}

			if want := tc.ID(); !got.Equal(want) {
				t.Logf("got:  %q\nwant: %q", got.String(), want.String())
				t.Fail()
			}
		})
	}
}

func TestFileSystem_WriteBranch(t *testing.T) {
	var ctx context.Context
	deadline, ok := t.Deadline()
	if ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	testLoc, err := os.MkdirTemp("", "envelopes")
	if err != nil {
		t.Error(err)
		return
	}

	subject := filesystem.FileSystem{
		Root: testLoc,
	}

	expected := envelopes.Transaction{
		Comment: "Freedom is secured not by the fulfilling of one's desires, but by the removal of desire... No man is free who is not master of himself.",
	}.ID()

	err = subject.WriteBranch(ctx, persist.DefaultBranch, expected)
	if err != nil {
		t.Error(err)
		return
	}

	got, err := subject.ReadBranch(ctx, persist.DefaultBranch)
	if err != nil {
		t.Error(err)
		return
	}

	if !got.Equal(expected) {
		t.Errorf("round trip failed.\n\tgot:  %s\n\twant: %s", got, expected)
	}
}

func TestFileSystem_RoundTrip_Attachment(t *testing.T) {
	var ctx context.Context
	deadline, ok := t.Deadline()
	if ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline)
		defer cancel()
	}

	fishingRegulations := envelopes.Attachment{
		Extension: "pdf",
		Comment:   "\"There's always a bigger fish\" -Qui-Gon Jinn",
		Contents: func(ctx context.Context) (io.ReadCloser, error) {
			return os.Open("../../2025ORFW.pdf")
		},
	}

	var err error
	fishingRegulations.ContentID, err = fishingRegulations.ContentSHA1(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	testDir := filepath.Join("testdata", "test", "filesystem", "roundtripAttachment")
	err = os.MkdirAll(testDir, os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		err := os.RemoveAll(testDir)
		if err != nil {
			t.Logf("unable to delete %q: %v", testDir, err)
		}
	}()

	repo, err := filesystem.OpenRepository(ctx, testDir)
	if err != nil {
		t.Error(err)
		return
	}

	err = repo.WriteAttachment(ctx, fishingRegulations)
	if err != nil {
		t.Error(err)
		return
	}

	var rehydrated envelopes.Attachment
	err = repo.LoadAttachment(ctx, fishingRegulations.ID(), &rehydrated)
	if err != nil {
		t.Error(err)
		return
	}

	var got envelopes.ID
	got, err = rehydrated.ContentSHA1(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	if !got.Equal(fishingRegulations.ContentID) {
		t.Error("actual contents of the rehydrated file does not match original")
	}
}

func TestFileSystem_TransactionRoundTrip(t *testing.T) {
	var err error
	var ctx context.Context
	deadline, ok := t.Deadline()
	if ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	fishingRegulations := envelopes.Attachment{
		Extension: "pdf",
		Comment:   "\"There's always a bigger fish\" -Qui-Gon Jinn",
		Contents: func(ctx context.Context) (io.ReadCloser, error) {
			return os.Open("../../2025ORFW.pdf")
		},
	}

	fishingRegulations.ContentID, err = fishingRegulations.ContentSHA1(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	testCases := []envelopes.Transaction{
		{},
		{
			State: &envelopes.State{
				Budget: &envelopes.Budget{
					Balance: envelopes.Balance{"USD": big.NewRat(100, 1)},
					Children: map[string]*envelopes.Budget{
						"groceries": {
							Balance: envelopes.Balance{"USD": big.NewRat(15000, 1)},
						},
					},
				},
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(15100, 1)},
				},
			},
		},
		{
			Reverts: []envelopes.ID{envelopes.Transaction{Comment: "Not very good"}.ID()},
			State: &envelopes.State{
				Budget: &envelopes.Budget{
					Balance: envelopes.Balance{"USD": big.NewRat(100, 1)},
					Children: map[string]*envelopes.Budget{
						"groceries": {
							Balance: envelopes.Balance{"USD": big.NewRat(15000, 1)},
						},
					},
				},
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(15100, 1)},
				},
			},
		},
		{
			State: &envelopes.State{
				Budget: &envelopes.Budget{
					Balance: envelopes.Balance{"USD": big.NewRat(100, 1)},
					Children: map[string]*envelopes.Budget{
						"groceries": {
							Balance: envelopes.Balance{"USD": big.NewRat(15000, 1)},
						},
					},
				},
				Accounts: envelopes.Accounts{
					"checking": envelopes.Balance{"USD": big.NewRat(15100, 1)},
				},
			},
			Attachments: map[string]envelopes.Attachment{
				"2025ODFW.pdf": fishingRegulations,
			},
		},
	}

	testDir := filepath.Join("testdata", "test", "filesystem", "roundtrip")
	err = os.MkdirAll(testDir, os.ModePerm)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		err := os.RemoveAll(testDir)
		if err != nil {
			t.Logf("unable to delete %q: %v", testDir, err)
		}
	}()

	repo, err := filesystem.OpenRepository(ctx, testDir)
	if err != nil {
		t.Error(err)
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%T/%s", tc, tc.ID()), func(t *testing.T) {

			err := repo.WriteTransaction(ctx, tc)
			if err != nil {
				t.Error(err)
				return
			}

			var loaded envelopes.Transaction
			err = repo.LoadTransaction(ctx, tc.ID(), &loaded)
			if err != nil {
				t.Error(err)
				return
			}

			got := loaded.ID()
			want := tc.ID()
			if got != want {
				t.Logf("\ngot: %q\nwant: %q", got, want)
				t.Fail()
			}
		})
	}
}

func TestFileSystem_ListBranches(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	testCases := []struct {
		location string
		expected []string
	}{
		{filepath.Join(".", "testdata", "test4", ".baronial"), []string{"backup", "main"}},
		{filepath.Join(".", "testdata", "test3", ".baronial"), []string{}},
	}

	subject := &filesystem.FileSystem{}

	for _, tc := range testCases {
		t.Run(tc.location, func(t *testing.T) {
			ctx2, cancel2 := context.WithTimeout(ctx, 100*time.Second)
			defer cancel2()

			subject.Root = tc.location

			branches, err := subject.ListBranches(ctx2)
			if err != nil {
				t.Error(err)
				return
			}

			i := 0
			for got := range branches {

				if i >= len(tc.expected) {
					t.Logf("Too many elements encountered, example: %s", got)
					t.Fail()
					return
				}

				if got != tc.expected[i] {
					t.Logf("\n\tat position %d\n\tgot:  %q\n\twant: %q", i, got, tc.expected[i])
					t.Fail()
					return
				}
				i++
			}

			if i != len(tc.expected) {
				t.Logf("Too few results, got %d want %d", i, len(tc.expected))
				t.Fail()
			}
		})
	}
}

func BenchmarkFileSystem_RoundTrip(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	benchDir := filepath.Join("testdata", "bench", "filesystem", "roundtrip")
	err := os.MkdirAll(benchDir, os.ModePerm)
	if err != nil {
		b.Log(err)
		b.FailNow()
	}
	defer func() {
		err = os.RemoveAll(benchDir)
		if err != nil {
			b.Logf("unable to clean up %q: %v", benchDir, err)
		}
	}()

	repo, err := filesystem.OpenRepository(ctx, benchDir)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		currentBudget := envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(int64(i), 1)}}
		err = repo.WriteBudget(ctx, currentBudget)
		if err != nil {
			b.Error(err)
			return
		}
		var loaded envelopes.Budget
		err = repo.LoadBudget(ctx, currentBudget.ID(), &loaded)
		if err != nil {
			b.Error(err)
			return
		}
	}
	b.StopTimer()
}

func TestFileSystem_WriteTag(t *testing.T) {
	var ctx context.Context
	deadline, ok := t.Deadline()
	if ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	testLoc, err := os.MkdirTemp("", "envelopes")
	if err != nil {
		t.Error(err)
		return
	}

	subject := filesystem.FileSystem{
		Root: testLoc,
	}

	expectedID := envelopes.Transaction{
		Comment: "Tags are like branches, but they don't advance or anything.",
	}.ID()
	expectedComment := "Release 1.0.0\nFirst stable release"

	expectedTag := persist.Tag{
		ID:      expectedID,
		Comment: expectedComment,
	}

	err = subject.WriteTag(ctx, "v1.0.0", expectedTag)
	if err != nil {
		t.Error(err)
		return
	}

	got, err := subject.ReadTag(ctx, "v1.0.0")
	if err != nil {
		t.Error(err)
		return
	}

	if got.ID != expectedTag.ID {
		t.Logf("\ngot ID:  %X\nwant ID: %X", got.ID, expectedTag.ID)
		t.Fail()
	}

	if got.Comment != expectedTag.Comment {
		t.Logf("\ngot comment:  %q\nwant comment: %q", got.Comment, expectedTag.Comment)
		t.Fail()
	}
}

func TestFileSystem_ListTags(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	testCases := []struct {
		location string
		expected []string
	}{
		{filepath.Join(".", "testdata", "test5", ".baronial"), []string{"v1.0.0", "v1.1.0"}},
		{filepath.Join(".", "testdata", "test3", ".baronial"), []string{}},
	}

	subject := &filesystem.FileSystem{}

	for _, tc := range testCases {
		t.Run(tc.location, func(t *testing.T) {
			ctx2, cancel2 := context.WithTimeout(ctx, 100*time.Second)
			defer cancel2()

			subject.Root = tc.location

			tags, err := subject.ListTags(ctx2)
			if err != nil {
				t.Error(err)
				return
			}

			i := 0
			for got := range tags {

				if i >= len(tc.expected) {
					t.Logf("Too many elements encountered, example: %s", got)
					t.Fail()
					return
				}

				if got != tc.expected[i] {
					t.Logf("\n\tat position %d\n\tgot:  %q\n\twant: %q", i, got, tc.expected[i])
					t.Fail()
					return
				}
				i++
			}

			if i != len(tc.expected) {
				t.Logf("Too few results, got %d want %d", i, len(tc.expected))
				t.Fail()
			}
		})
	}
}
