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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/marstr/envelopes"
)

func Test_Resolve(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var test3and4Transactions [4]envelopes.ID
	test3and4Transactions[0].UnmarshalText([]byte("f3bd1757f2ccfe63eefce93ffe4c046166d0a1a5"))
	test3and4Transactions[1].UnmarshalText([]byte("c76ece80f16ef8cfe0213ca4b568d7d576a96f9d"))
	test3and4Transactions[2].UnmarshalText([]byte("d9ff31f511c1fce2ec75b5efc500a6dfb4d83452"))
	test3and4Transactions[3].UnmarshalText([]byte("434926f0aa5dde53c24c81e8061f99ab963bb98f"))

	testCases := []struct {
		repoLocation string
		subject      RefSpec
		expected     envelopes.ID
	}{
		{"./testdata/test1", "HEAD", envelopes.ID{}},
		{"./testdata/test2", "HEAD", envelopes.ID{}},
		{"./testdata/test3/.baronial", "HEAD", test3and4Transactions[3]},
		{"./testdata/test3/.baronial", "HEAD^", test3and4Transactions[2]},
		{"./testdata/test3/.baronial", "HEAD~1", test3and4Transactions[2]},
		{"./testdata/test3/.baronial", "HEAD^^", test3and4Transactions[1]},
		{"./testdata/test3/.baronial", "HEAD~3", test3and4Transactions[0]},
		{"./testdata/test3/.baronial", "d9ff31f511c1fce2ec75b5efc500a6dfb4d83452^", test3and4Transactions[1]},
		{"./testdata/test3/.baronial", "434926f0aa5dde53c24c81e8061f99ab963bb98f~3", test3and4Transactions[0]},
		{"./testdata/test4/.baronial", "backup", test3and4Transactions[1]},
		{"./testdata/test4/.baronial", "master", test3and4Transactions[3]},
		{"./testdata/test4/.baronial", "master~2", test3and4Transactions[1]},
		{"./testdata/test4/.baronial", "HEAD", test3and4Transactions[3]},
	}

	fs := &FileSystem{}

	loader := struct{
		Loader
		CurrentReader
		BranchReader
		BranchLister
	}{
		Loader: &DefaultLoader{Fetcher: fs},
		CurrentReader: fs,
		BranchLister: fs,
		BranchReader: fs,
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%q", tc.repoLocation, string(tc.subject)), func(t *testing.T) {
			fs.Root = tc.repoLocation
			got, err := Resolve(ctx, loader, tc.subject)
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
