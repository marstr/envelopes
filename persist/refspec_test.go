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
	const primaryBranch = DefaultBranch
	const secondaryBranch = "backup"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockRepo := NewMockRepository(2, 4)

	var transactions [4]envelopes.Transaction
	transactions[0] = envelopes.Transaction{
		Comment: "A!",
	}
	aid := transactions[0].ID()
	err := mockRepo.Write(ctx, transactions[0])
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
	err = mockRepo.Write(ctx, transactions[1])
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
	err = mockRepo.Write(ctx, transactions[2])
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
	err = mockRepo.Write(ctx, transactions[3])
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
