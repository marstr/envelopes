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

package persist_test

import (
	"context"
	"fmt"

	"math/big"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

func TestFileSystem_Current(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testCases := []string{
		"./testdata/test1",
		"./testdata/test2",
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			subject := persist.FileSystem{Root: tc}
			resolver := persist.RefSpecResolver{
				Loader:        persist.DefaultLoader{Fetcher: subject},
				Brancher:      subject,
				CurrentReader: subject,
			}
			head, err := subject.Current(context.Background())
			if err != nil {
				t.Error(err)
			}

			headId, err := resolver.Resolve(ctx, head)

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

	testLocation := path.Join("testdata", "test", "roundtrip", "current")
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

	subject := persist.FileSystem{Root: testLocation}
	writer := persist.DefaultWriter{
		Stasher: subject,
	}
	resolver := persist.RefSpecResolver{
		Loader:        persist.DefaultLoader{Fetcher: subject},
		Brancher:      subject,
		CurrentReader: subject,
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			err := writer.Write(ctx, tc)
			if err != nil {
				t.Error(err)
				return
			}

			err = subject.WriteCurrent(ctx, tc)
			if err != nil {
				t.Error(err)
				return
			}

			refSpec, err := subject.Current(ctx)
			if err != nil {
				t.Error(err)
				return
			}

			got, err := resolver.Resolve(ctx, refSpec)
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

func TestFileSystem_TransactionRoundTrip(t *testing.T) {
	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()
	ctx := context.Background()

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
	}

	testDir := path.Join("testdata", "test", "filesystem", "roundtrip")
	err := os.MkdirAll(testDir, os.ModePerm)
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

	subject := persist.FileSystem{Root: testDir}
	writer := persist.DefaultWriter{Stasher: subject}
	reader := persist.DefaultLoader{Fetcher: subject}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%T/%s", tc, tc.ID()), func(t *testing.T) {

			err := writer.Write(ctx, tc)
			if err != nil {
				t.Error(err)
				return
			}

			var loaded envelopes.Transaction
			err = reader.Load(ctx, tc.ID(), &loaded)
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
		{"./testdata/test4/.baronial", []string{"backup", "master"}},
		{"./testdata/test3/.baronial", []string{}},
	}

	subject := &persist.FileSystem{}

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

func TestFileSystem_Clone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	original := persist.FileSystem{
		Root: path.Join(".", "testdata", "test3", ".baronial"),
	}
	resolver := persist.RefSpecResolver{
		Loader:        persist.DefaultLoader{Fetcher: original},
		Brancher:      original,
		CurrentReader: original,
	}

	outputLoc, err := ioutil.TempDir("", "envelopesCloneTest")
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("Output Location: %s\n", outputLoc)

	subject := persist.FileSystem{
		Root: outputLoc,
	}

	head, err := original.Current(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	headId, err := resolver.Resolve(ctx, head)
	if err != nil {
		t.Error(err)
		return
	}

	err = subject.Clone(ctx, headId, persist.DefaultBranch, persist.DefaultLoader{Fetcher: original})
	if err != nil {
		t.Error(err)
		return
	}

	branchCheck, cancelBranchEnumeration := context.WithCancel(ctx)
	expectedResults := []string{persist.DefaultBranch}
	results, err := subject.ListBranches(branchCheck)
	if err != nil {
		t.Error(err)
		cancelBranchEnumeration()
		return
	}

	encounteredUnexpected := false
	encountered := 0
	for entry := range results {
		encountered++
		if len(expectedResults) > 0 {
			if expectedResults[0] != entry {
				encounteredUnexpected = true
				t.Fail()
			}
		} else {
			t.Log("extra branches encountered")
			t.Fail()
			break
		}
	}
	cancelBranchEnumeration()

	if encounteredUnexpected {
		t.Logf("Unexpected branch results encountered")
	}
	if encountered < len(expectedResults) {
		t.Logf("Too few branches encountered")
		t.Fail()
	}
}

func TestFileSystem_WriteCurrent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("from-scratch", testFileSystem_WriteCurrentFromScratch(ctx))
}

func testFileSystem_WriteCurrentFromScratch(ctx context.Context) func(t *testing.T) {
	return func(t *testing.T) {
		tempDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Error(err)
			return
		}

		t.Logf("Repo Location: %s", tempDir)

		subject := persist.FileSystem{
			Root: tempDir,
		}
		resolver := persist.RefSpecResolver{
			Loader:        persist.DefaultLoader{Fetcher: subject},
			Brancher:      subject,
			CurrentReader: subject,
		}

		firstTransaction := envelopes.Transaction{
			Merchant: "Aer Lingus",
			Time:     time.Now(),
			Comment:  "foo",
		}

		err = subject.WriteBranch(ctx, persist.DefaultBranch, firstTransaction.ID())
		if err != nil {
			t.Error(err)
			return
		}

		err = subject.SetCurrent(ctx, persist.DefaultBranch)
		if err != nil {
			t.Error(err)
			return
		}

		err = subject.WriteCurrent(ctx, firstTransaction)
		if err != nil {
			t.Error(err)
			return
		}

		actualRef, err := subject.Current(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		if actualRef != persist.DefaultBranch {
			t.Logf("Unexpected RefSpec!\n\twant:\t%q\n\tgot: \t%q", persist.DefaultBranch, actualRef)
			t.Fail()
		}

		want := firstTransaction.ID()
		got, err := subject.Current(ctx)
		if err != nil {
			t.Error(err)
			return
		}

		gotId, err := resolver.Resolve(ctx, got)
		if err != nil {
			t.Error(err)
			return
		}

		if !gotId.Equal(want) {
			t.Logf("Unexpected Transaction ID!\n\twant:\t%q\n\tgot: \t%q", gotId.String(), want.String())
			t.Fail()
		}
	}
}

func BenchmarkFileSystem_CloneSmall(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	original := persist.FileSystem{
		Root: path.Join(".", "testdata", "test3", ".baronial"),
	}
	resolver := persist.RefSpecResolver{
		Loader:        persist.DefaultLoader{Fetcher: original},
		Brancher:      original,
		CurrentReader: original,
	}
	head, err := original.Current(ctx)
	if err != nil {
		b.Error(err)
		return
	}
	headId, err := resolver.Resolve(ctx, head)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		outputLoc, err := ioutil.TempDir("", "cloneSmallBenchmark")
		if err != nil {
			b.Error(err)
			return
		}

		subject := persist.FileSystem{
			Root: outputLoc,
		}

		b.StartTimer()
		err = subject.Clone(ctx, headId, persist.DefaultBranch, persist.DefaultLoader{Fetcher: original})
		if err != nil {
			b.Error(err)
			return
		}
	}
}

func BenchmarkFileSystem_RoundTrip(b *testing.B) {
	benchDir := path.Join("testdata", "bench", "filesystem", "roundtrip")
	subject := persist.FileSystem{Root: benchDir}
	writer := persist.DefaultWriter{Stasher: subject}
	reader := persist.DefaultLoader{Fetcher: subject}
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		currentBudget := envelopes.Budget{Balance: envelopes.Balance{"USD": big.NewRat(int64(i), 1)}}
		err = writer.Write(context.Background(), currentBudget)
		if err != nil {
			b.Error(err)
			return
		}
		var loaded envelopes.Budget
		err = reader.Load(context.Background(), currentBudget.ID(), &loaded)
		if err != nil {
			b.Error(err)
			return
		}
	}
	b.StopTimer()
}
