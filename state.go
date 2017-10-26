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

package envelopes

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
)

// State captures the values of all Budgets and Accounts.
type State struct {
	Budget
	Accounts []Account
	ParentID [20]byte
}

// MarshalJSON converts a State loaded in memory into a persistable/transmittable string in the form of a JSON object.
func (s State) MarshalJSON() ([]byte, error) {
	var err error
	results := new(bytes.Buffer)

	results.WriteRune('{')

	marshaledBudget, err := json.Marshal(s.Budget)
	if err != nil {
		return nil, err
	}
	fmt.Fprint(results, `"budget":`)
	fmt.Fprint(results, string(marshaledBudget))
	results.WriteRune(',')

	marshaledAccounts, err := json.Marshal(s.Accounts)
	if err != nil {
		return nil, err
	}

	fmt.Fprint(results, `"accounts":`)
	fmt.Fprint(results, string(marshaledAccounts))
	results.WriteRune(',')

	fmt.Fprint(results, `"parent":`)
	fmt.Fprintf(results, `"%x"`, s.ParentID)

	results.WriteRune('}')

	return results.Bytes(), nil
}

// ID fetches the identifier associated with this `State`.
func (s State) ID() [20]byte {
	message, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return sha1.Sum(message)
}
