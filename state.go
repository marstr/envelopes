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
	"fmt"
)

// State captures the values of all Budgets and Accounts.
type State struct {
	Budget   *Budget
	Accounts Accounts
}

// ID calculates the SHA1 hash of this object.
func (s State) ID() (id ID) {
	marshaled, err := s.MarshalText()
	if err != nil {
		return ID{}
	}
	return sha1.Sum(marshaled)
}

func (s State) MarshalText() ([]byte, error) {

	identityBuilder := identityBuilders.Get().(*bytes.Buffer)
	identityBuilder.Reset()
	defer identityBuilders.Put(identityBuilder)

	if s.Budget == nil {
		s.Budget = &Budget{}
	}
	_, err := fmt.Fprintf(identityBuilder, "budget %s\n", s.Budget.ID())
	if err != nil {
		return nil, err
	}

	if s.Accounts == nil {
		s.Accounts = make(Accounts, 0)
	}
	_, err = fmt.Fprintf(identityBuilder, "accounts %s\n", s.Accounts.ID())
	if err != nil {
		return nil, err
	}

	return identityBuilder.Bytes(), nil
}
