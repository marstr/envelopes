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

func TestRefSpecResolver_Resolve(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var test3and4Transactions [4]envelopes.ID
	test3and4Transactions[0].UnmarshalText([]byte("7a41c93dd3f21d4a7cc4513451a5706e2f89421b"))
	test3and4Transactions[1].UnmarshalText([]byte("1396f229bbc973c6c10087d68663fbcdac0b8fae"))
	test3and4Transactions[2].UnmarshalText([]byte("e26999263bdac65fabfa65ca99a53bf16688dced"))
	test3and4Transactions[3].UnmarshalText([]byte("558a04a94c1566aaaba9ee18d33d19966a6cf254"))

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
		{"./testdata/test3/.baronial", "e26999263bdac65fabfa65ca99a53bf16688dced^", test3and4Transactions[1]},
		{"./testdata/test3/.baronial", "558a04a94c1566aaaba9ee18d33d19966a6cf254~3", test3and4Transactions[0]},
		{"./testdata/test4/.baronial", "backup", test3and4Transactions[1]},
		{"./testdata/test4/.baronial", "master", test3and4Transactions[3]},
		{"./testdata/test4/.baronial", "master~2", test3and4Transactions[1]},
		{"./testdata/test4/.baronial", "HEAD", test3and4Transactions[3]},
	}

	fs := &FileSystem{}

	resolver := RefSpecResolver{
		Loader: DefaultLoader{
			Fetcher: fs,
		},
		Brancher: fs,
		Fetcher:  fs,
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%q", tc.repoLocation, string(tc.subject)), func(t *testing.T) {
			fs.Root = tc.repoLocation
			got, err := resolver.Resolve(ctx, tc.subject)
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
