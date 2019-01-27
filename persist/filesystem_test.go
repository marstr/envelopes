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
	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
	"os"
	"path"
	"testing"
)

func TestFileSystem_Current(t *testing.T) {
	testCases := []string{
		"./testdata/test1",
		"./testdata/test2",
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			subject := persist.FileSystem{Root: tc}
			id, err := subject.Current(context.Background())
			if err != nil {
				t.Error(err)
			}

			for i := range id {
				if id[i] != 0 {
					t.Logf("got: %X want: %X", id, envelopes.ID{})
					t.Fail()
					break
				}
			}
		})
	}
}

func TestFileSystem_RoundTrip_Current(t *testing.T) {
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
		{Amount: 1729},
	}

	subject := persist.FileSystem{Root: testLocation}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			err := subject.WriteCurrent(context.Background(), &tc)
			if err != nil {
				t.Error(err)
				return
			}

			got, err := subject.Current(context.Background())
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

func TestFileSystem_RoundTrip(t *testing.T) {
	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()
	ctx := context.Background()

	testCases := []envelopes.Transaction{
		{},
		{
			State: &envelopes.State{
				Budget: &envelopes.Budget{
					Balance: 100,
					Children: map[string]*envelopes.Budget{
						"groceries": {
							Balance: 15000,
						},
					},
				},
				Accounts: envelopes.Accounts{
					"checking": 15100,
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
		currentBudget := envelopes.Budget{Balance: envelopes.Balance(i)}
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
