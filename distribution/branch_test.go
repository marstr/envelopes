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

package distribution_test

import (
	"fmt"
	"testing"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/distribution"
	"github.com/marstr/envelopes/evaluate"
)

func ExampleBranch_Distribute() {
	subject := distribution.Branch{
		Affirmative: (*distribution.Identity)(&envelopes.Budget{Name: "Groceries"}),
		Negative:    (*distribution.Identity)(&envelopes.Budget{Name: "Transit"}),
		Decision:    evaluate.Bool(true),
	}

	fmt.Println(subject.Distribute(42))
	// Output: ["Groceries":$0.42]
}

func TestBranch_Distribution(t *testing.T) {
	testCases := []struct {
		distribution.Branch
		amount int64
		want   envelopes.Effect
	}{}

	for _, tc := range testCases {
		got := tc.Distribute(tc.amount)

		if !got.Equal(tc.want) {
			t.Logf("\ngot:  %s\nwant: %s", got, tc.want)
			t.Fail()
		}
	}
}
