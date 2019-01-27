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

package envelopes_test

import (
	"fmt"
	"testing"
)
import "github.com/marstr/envelopes"

func TestState_ID_Deterministic(t *testing.T) {
	testCases := []envelopes.State{
		{},
		{Budget: &envelopes.Budget{Balance: 1729}},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%x", tc.ID()), func(t *testing.T) {
			first := tc.ID()
			t.Logf("First ID: %s", first)
			for i := 0; i < 30; i++ {
				subsequent := tc.ID()

				for j := 0; j < len(first); j++ {
					if first[j] != subsequent[j] {
						t.Logf("Subsequent ID: %s", subsequent)
						t.FailNow()
					}
				}
			}
		})
	}
}
