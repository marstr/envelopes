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
)

// State captures the values of all Budgets and Accounts.
type State struct {
	budget   ID
	accounts ID
}

// ID calculates the SHA1 hash of this object.
func (s State) ID() (calculated ID) {
	marshaled, err := json.Marshal(s)
	if err != nil {
		return
	}
	calculated = sha1.Sum(marshaled)
	return
}

// Budget fetches the instance of `envelopes.Budget` associated with this
// State.
func (s State) Budget() ID {
	return s.budget
}

// SetBudget creates a copy of the State with the updated Budget.
func (s State) SetBudget(id ID) State {
	s.budget = id
	return s
}

// Accounts fetches a list of each Account and its balance that is associated
// with this State.
func (s State) Accounts() ID {
	return s.accounts
}

// SetAccounts creates a copy of the Stae with the updated Accounts.
func (s State) SetAccounts(id ID) State {
	s.accounts = id
	return s
}

// MarshalJSON creates a JSON representation of this State.
func (s State) MarshalJSON() ([]byte, error) {
	var err error

	marshaledAccountID, err := json.Marshal(s.Accounts())
	if err != nil {
		return nil, err
	}

	marshaledBudgetID, err := json.Marshal(s.Budget())
	if err != nil {
		return nil, err
	}

	builder := bytes.Buffer{}

	builder.WriteRune('{')

	builder.WriteString(`"accounts":`)
	builder.Write(marshaledAccountID)

	builder.WriteRune(',')

	builder.WriteString(`"budget":`)
	builder.Write(marshaledBudgetID)

	builder.WriteRune('}')

	return builder.Bytes(), nil
}

// UnmarshalJSON populates a State as marked up by the JSON object provided.
func (s *State) UnmarshalJSON(content []byte) (err error) {
	var intermediate map[string]json.RawMessage

	var contender State

	err = json.Unmarshal(content, &intermediate)
	if err != nil {
		return
	}

	err = json.Unmarshal(intermediate["accounts"], &contender.accounts)
	if err != nil {
		return
	}

	err = json.Unmarshal(intermediate["budget"], &contender.budget)
	if err != nil {
		return
	}

	*s = contender
	return
}
