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

// ID is a SHA1 based on the JSON marshaling of a State object.
type ID [20]byte

// MarshalJSON creates a JSON string populated with the hex value of the identifier.
func (id ID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%x\"", id)), nil
}

// UnmarshalJSON retreives a 20-byte SHA1 hash from a JSON string.
func (id *ID) UnmarshalJSON(contents []byte) error {
	_, err := fmt.Fscanf(bytes.NewReader(contents), "\"%x\"", id)
	return err
}

// State captures the values of all Budgets and Accounts.
type State struct {
	Budget   `json:"budget"`
	Accounts []Account `json:"accounts"`
	Parent   ID        `json:"parent"`
}

// ID fetches the identifier associated with this `State`.
func (s State) ID() ID {
	message, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return sha1.Sum(message)
}
