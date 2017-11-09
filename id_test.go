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
	"testing"

	"github.com/marstr/envelopes"
)

func TestID_MarshalText(t *testing.T) {
	testCases := []struct {
		envelopes.ID
		expected string
	}{
		{envelopes.ID{}, "0000000000000000000000000000000000000000"},
	}

	for _, tc := range testCases {
		t.Run(tc.String(), func(t *testing.T) {
			got, err := tc.ID.MarshalText()
			if err != nil {
				t.Error(err)
			}

			if string(got) != tc.expected {
				t.Logf("\ngot:  %q\nwant: %q", string(got), tc.expected)
				t.Fail()
			}
		})
	}
}
