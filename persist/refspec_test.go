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

package persist

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/marstr/envelopes"
)

func TestResolutionDiagnostics(t *testing.T) {
	ctx := context.Background()
	mock := NewMockRepository(1, 10)
	parent := envelopes.Transaction{Comment: "private-parent-comment-sentinel"}
	child := envelopes.Transaction{Comment: "private-child-comment-sentinel", Parents: []envelopes.ID{parent.ID()}}
	for _, transaction := range []envelopes.Transaction{parent, child} {
		if err := mock.WriteTransaction(ctx, transaction); err != nil {
			t.Fatal(err)
		}
	}
	if err := mock.WriteBranch(ctx, DefaultBranch, child.ID()); err != nil {
		t.Fatal(err)
	}
	if err := mock.WriteTag(ctx, "tag", Tag{ID: parent.ID(), Comment: "private-tag-comment-sentinel"}); err != nil {
		t.Fatal(err)
	}
	if err := mock.SetCurrent(ctx, RefSpec(DefaultBranch)); err != nil {
		t.Fatal(err)
	}
	missing := envelopes.ID{99}
	for _, bare := range []bool{false, true} {
		for _, tc := range []struct {
			name    string
			ref     RefSpec
			stage   string
			id      envelopes.ID
			wantErr error
		}{
			{name: "transaction", ref: RefSpec(child.ID().String()), stage: "transaction", id: child.ID()},
			{name: "uninitialized", ref: RefSpec((envelopes.ID{}).String()), stage: "transaction"},
			{name: "branch", ref: RefSpec(DefaultBranch), stage: "branch", id: child.ID()},
			{name: "tag", ref: "tag", stage: "tag", id: parent.ID()},
			{name: "head", ref: MostRecentTransactionAlias, stage: "head", id: child.ID()},
			{name: "caret", ref: RefSpec(DefaultBranch) + "^", stage: "caret", id: parent.ID()},
			{name: "tilde", ref: RefSpec(DefaultBranch) + "~1", stage: "tilde", id: parent.ID()},
			{name: "invalid", ref: "unknown", stage: "unresolved", wantErr: ErrNoRefSpec("unknown")},
			{name: "missing object", ref: RefSpec(missing.String()), stage: "transaction", wantErr: ErrObjectNotFound(missing)},
		} {
			t.Run(fmt.Sprintf("%s/bare=%v", tc.name, bare), func(t *testing.T) {
				var logs bytes.Buffer
				repo := &loggingRepository{
					MockRepository: mock,
					logger:         slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})).With("command", "resolve"),
				}
				var id envelopes.ID
				var err error
				operation := "resolve"
				if bare {
					operation = "bare_resolve"
					if tc.name == "head" {
						tc.stage, tc.id, tc.wantErr = "unresolved", envelopes.ID{}, ErrNoRefSpec(tc.ref)
					}
					id, err = BareResolve(ctx, repo, tc.ref)
				} else {
					id, err = Resolve(ctx, repo, tc.ref)
				}
				if err != tc.wantErr || id != tc.id {
					t.Fatalf("resolution changed: got %s, %v; want %s, %v", id, err, tc.id, tc.wantErr)
				}
				records := diagnosticRecords(t, &logs)
				record := findDiagnostic(t, records, operation, tc.stage)
				if record["reference"] != string(tc.ref) || record["command"] != "resolve" {
					t.Fatalf("missing resolution context: %v", record)
				}
				if err == nil && record["object_id"] != id.String() {
					t.Fatalf("missing resolved ID: %v", record)
				}
				if tc.name == "tag" {
					probe := findDiagnostic(t, records, operation, "branch_probe")
					if probe["error"] == nil {
						t.Fatal("failed branch probe lost its cause")
					}
				}
				for _, record := range records {
					if record["level"] != "DEBUG" {
						t.Fatalf("failed probe produced warning/error: %v", record)
					}
				}
				for _, private := range []string{parent.Comment, child.Comment, "private-tag-comment-sentinel"} {
					if strings.Contains(logs.String(), private) {
						t.Fatal("logged object contents during resolution")
					}
				}
			})
		}
	}
}

func TestResolutionDiagnosticsDisabled(t *testing.T) {
	ctx := context.Background()
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	previous := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(previous) })
	mock := NewMockRepository(1, 10)
	for _, repo := range []RepositoryReader{
		mock,
		&loggingRepository{MockRepository: mock},
		&loggingRepository{
			MockRepository: mock,
			logger:         slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelInfo})),
		},
	} {
		if _, err := Resolve(ctx, repo, "unknown"); err != ErrNoRefSpec("unknown") {
			t.Fatal(err)
		}
		if _, err := BareResolve(ctx, repo, "unknown"); err != ErrNoRefSpec("unknown") {
			t.Fatal(err)
		}
	}
	if logs.Len() != 0 {
		t.Fatal("disabled resolution diagnostics emitted logs")
	}
}

func TestResolve(t *testing.T) {
	const primaryBranch = DefaultBranch
	const secondaryBranch = "backup"

	ctx := context.Background()

	mockRepo := NewMockRepository(2, 4)

	var transactions [4]envelopes.Transaction
	transactions[0] = envelopes.Transaction{
		Comment: "A!",
	}
	aid := transactions[0].ID()
	err := mockRepo.WriteTransaction(ctx, transactions[0])
	if err != nil {
		t.Error(err)
		return
	}

	transactions[1] = envelopes.Transaction{
		Comment: "B!",
		Parents: []envelopes.ID{
			aid,
		},
	}
	bid := transactions[1].ID()
	err = mockRepo.WriteTransaction(ctx, transactions[1])
	if err != nil {
		t.Error(err)
		return
	}
	err = mockRepo.WriteBranch(ctx, secondaryBranch, bid)
	if err != nil {
		t.Error(err)
		return
	}

	transactions[2] = envelopes.Transaction{
		Comment: "C!",
		Parents: []envelopes.ID{
			bid,
		},
	}
	cid := transactions[2].ID()
	err = mockRepo.WriteTransaction(ctx, transactions[2])
	if err != nil {
		t.Error(err)
		return
	}

	transactions[3] = envelopes.Transaction{
		Comment: "D!",
		Parents: []envelopes.ID{
			cid,
		},
	}
	did := transactions[3].ID()
	err = mockRepo.WriteTransaction(ctx, transactions[3])
	if err != nil {
		t.Error(err)
		return
	}

	err = mockRepo.WriteBranch(ctx, primaryBranch, did)
	if err != nil {
		t.Error(err)
		return
	}

	err = mockRepo.SetCurrent(ctx, primaryBranch)
	if err != nil {
		t.Error(err)
		return
	}

	testCases := []struct {
		subject  RefSpec
		expected envelopes.ID
	}{
		{MostRecentTransactionAlias, did},
		{MostRecentTransactionAlias + "^", cid},
		{MostRecentTransactionAlias + "~1", cid},
		{MostRecentTransactionAlias + "^^", bid},
		{MostRecentTransactionAlias + "~3", aid},
		{RefSpec(cid.String() + "^"), bid},
		{RefSpec(did.String() + "~3"), aid},
		{secondaryBranch, bid},
		{primaryBranch, did},
		{primaryBranch + "~2", bid},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%q", string(tc.subject)), func(t *testing.T) {
			got, err := Resolve(ctx, mockRepo, tc.subject)
			if err != nil {
				t.Error(err)
				return
			}

			if !got.Equal(tc.expected) {
				t.Logf("\n\tgot:  %q\n\twant: %q", got, tc.expected)
				t.Fail()
			}
		})
	}
}

func TestBareResolve(t *testing.T) {
	const primaryBranch = DefaultBranch
	const secondaryBranch = "backup"

	ctx := context.Background()

	mockRepo := NewMockRepository(2, 4)

	var transactions [4]envelopes.Transaction
	transactions[0] = envelopes.Transaction{
		Comment: "A!",
	}
	aid := transactions[0].ID()
	err := mockRepo.WriteTransaction(ctx, transactions[0])
	if err != nil {
		t.Error(err)
		return
	}

	transactions[1] = envelopes.Transaction{
		Comment: "B!",
		Parents: []envelopes.ID{
			aid,
		},
	}
	bid := transactions[1].ID()
	err = mockRepo.WriteTransaction(ctx, transactions[1])
	if err != nil {
		t.Error(err)
		return
	}
	err = mockRepo.WriteBranch(ctx, secondaryBranch, bid)
	if err != nil {
		t.Error(err)
		return
	}

	transactions[2] = envelopes.Transaction{
		Comment: "C!",
		Parents: []envelopes.ID{
			bid,
		},
	}
	cid := transactions[2].ID()
	err = mockRepo.WriteTransaction(ctx, transactions[2])
	if err != nil {
		t.Error(err)
		return
	}

	transactions[3] = envelopes.Transaction{
		Comment: "D!",
		Parents: []envelopes.ID{
			cid,
		},
	}
	did := transactions[3].ID()
	err = mockRepo.WriteTransaction(ctx, transactions[3])
	if err != nil {
		t.Error(err)
		return
	}

	err = mockRepo.WriteBranch(ctx, primaryBranch, did)
	if err != nil {
		t.Error(err)
		return
	}
	testCases := []struct {
		subject  RefSpec
		expected envelopes.ID
	}{
		{RefSpec(cid.String() + "^"), bid},
		{RefSpec(did.String() + "~3"), aid},
		{secondaryBranch, bid},
		{primaryBranch, did},
		{primaryBranch + "~2", bid},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%q", string(tc.subject)), func(t *testing.T) {
			got, err := BareResolve(ctx, mockRepo, tc.subject)
			if err != nil {
				t.Error(err)
				return
			}

			if !got.Equal(tc.expected) {
				t.Logf("\n\tgot:  %q\n\twant: %q", got, tc.expected)
				t.Fail()
			}
		})
	}
}

func TestResolve_Tag(t *testing.T) {
	const primaryBranch = DefaultBranch
	const tagName = "v1.0.0"

	ctx := context.Background()

	mockRepo := NewMockRepository(2, 2)

	transaction := envelopes.Transaction{
		Comment: "Release 1.0.0",
	}
	tid := transaction.ID()
	err := mockRepo.WriteTransaction(ctx, transaction)
	if err != nil {
		t.Error(err)
		return
	}

	err = mockRepo.WriteBranch(ctx, primaryBranch, tid)
	if err != nil {
		t.Error(err)
		return
	}

	err = mockRepo.WriteTag(ctx, tagName, Tag{ID: tid, Comment: "Release 1.0.0"})
	if err != nil {
		t.Error(err)
		return
	}

	testCases := []struct {
		subject  RefSpec
		expected envelopes.ID
	}{
		{tagName, tid},
		{primaryBranch, tid},
		{RefSpec(tid.String()), tid},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%q", string(tc.subject)), func(t *testing.T) {
			got, err := Resolve(ctx, mockRepo, tc.subject)
			if err != nil {
				t.Error(err)
				return
			}

			if !got.Equal(tc.expected) {
				t.Logf("\n\tgot:  %q\n\twant: %q", got, tc.expected)
				t.Fail()
			}
		})
	}
}

func TestBareResolve_TagWhenNoBranch(t *testing.T) {
	// This test ensures tags are resolved even when a branch with the same name doesn't exist
	const tagName = "release"

	ctx := context.Background()

	mockRepo := NewMockRepository(2, 2)

	transaction := envelopes.Transaction{
		Comment: "Release transaction",
	}
	tid := transaction.ID()
	err := mockRepo.WriteTransaction(ctx, transaction)
	if err != nil {
		t.Error(err)
		return
	}

	// Create only a tag, no branch with this name
	err = mockRepo.WriteTag(ctx, tagName, Tag{ID: tid, Comment: "Release transaction"})
	if err != nil {
		t.Error(err)
		return
	}

	// Verify tag can be resolved
	got, err := BareResolve(ctx, mockRepo, RefSpec(tagName))
	if err != nil {
		t.Error(err)
		return
	}

	if !got.Equal(tid) {
		t.Logf("\n\tgot:  %q\n\twant: %q", got, tid)
		t.Fail()
	}
}
