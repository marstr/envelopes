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
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"
)

func TestFileSystem_RoundTrip(t *testing.T) {
	testCases := []envelopes.IDer{
		envelopes.Budget{},
		envelopes.State{},
		envelopes.Transaction{},
		envelopes.Budget{}.WithBalance(42),
		envelopes.Transaction{}.WithComment("This is only a test"),
	}

	testDir := path.Join("testdata", "test", "filesystem", "roundtrip")
	err := os.MkdirAll(testDir, os.ModePerm)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer func() {
		err := os.RemoveAll(testDir)
		if err != nil {
			t.Logf("unable to delete %q: %v", testDir, err)
		}
	}()

	subject := persist.FileSystem{
		Root: testDir,
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%T/%s", tc, tc.ID()), func(t *testing.T) {
			var err error
			switch tc.(type) {
			case envelopes.Budget:
				err = subject.WriteBudget(tc.(envelopes.Budget))
			case envelopes.State:
				err = subject.WriteState(tc.(envelopes.State))
			case envelopes.Transaction:
				err = subject.WriteTransaction(tc.(envelopes.Transaction))
			}
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			got, err := subject.Fetch(tc.ID())
			if err != nil {
				t.Error(err)
				t.FailNow()
			}

			expected, err := json.Marshal(tc)
			if string(got) != string(expected) {
				t.Logf("\ngot:  %q\nwant: %q", string(got), string(expected))
				t.Fail()
			}
		})
	}
}

func BenchmarkFileSystem_RoundTrip(b *testing.B) {
	benchDir := path.Join("testdata", "bench", "filesystem", "roundtrip")
	subject := persist.FileSystem{Root: benchDir}
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
		currentBudget := envelopes.Budget{}.WithBalance(int64(i))
		subject.WriteBudget(currentBudget)
		subject.Fetch(currentBudget.ID())
	}
	b.StopTimer()
}