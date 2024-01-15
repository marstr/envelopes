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
	"context"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"
	"time"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
	"github.com/marstr/envelopes/persist/filesystem"
)

func TestFileSystem_Current(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testCases := []string{
		"./testdata/test1",
		"./testdata/test2",
	}

	repo, err := filesystem.OpenRepository(ctx, "")
	if err != nil {
		t.Error(err)
	}
	subject := &repo.FileSystem

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			subject.Root = tc

			head, err := subject.Current(context.Background())
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
			err = repo.Load(ctx, tc.ID(), &loaded)
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

	benchDir := path.Join("testdata", "bench", "filesystem", "roundtrip")
	repo, err := filesystem.OpenRepository(ctx, benchDir)
	if err != nil {
		b.Error(err)
	}

	err = os.MkdirAll(benchDir, os.ModePerm)
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
		err = repo.WriteBudget(context.Background(), currentBudget)
		if err != nil {
			b.Error(err)
			return
		}
		var loaded envelopes.Budget
		err = repo.Load(context.Background(), currentBudget.ID(), &loaded)
		if err != nil {
			b.Error(err)
			return
		}
	}
	b.StopTimer()
}
