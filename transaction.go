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
	"time"
)

// Transaction represents one exchange of funds and how it impacted budgets.
type Transaction struct {
	State       *State
	ActualTime  time.Time
	PostedTime  time.Time
	EnteredTime time.Time
	Amount      Balance
	Merchant    string
	Committer   User
	Comment     string
	Parent      ID
}

// ID fetches a SHA1 hash of this object that will uniquely identify it.
func (t Transaction) ID() (id ID) {
	marshaled, err := t.MarshalText()
	if err != nil {
		return ID{}
	}
	return sha1.Sum(marshaled)
}

func (t Transaction) String() string {
	marshaled, err := t.MarshalText()
	if err != nil {
		return ""
	}

	return string(marshaled)
}

// MarshalText computes a string which uniquely represents this Transaction.
func (t Transaction) MarshalText() ([]byte, error) {
	const timeFormat = time.RFC3339
	identityBuilder := identityBuilders.Get().(*bytes.Buffer)
	identityBuilder.Reset()
	defer identityBuilders.Put(identityBuilder)
	defaultTime := time.Time{}

	if t.State == nil {
		t.State = &State{}
	}

	var err error
	_, err = fmt.Fprintf(identityBuilder, "state %s\n", t.State.ID())
	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintf(identityBuilder, "parent %s\n", t.Parent)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(identityBuilder, "posted time %s\n", t.PostedTime.Format(timeFormat))
	if err != nil {
		return nil, err
	}
	if t.ActualTime != defaultTime {
		_, err = fmt.Fprintf(identityBuilder, "actual time %s\n", t.ActualTime.Format(timeFormat))
		if err != nil {
			return nil, err
		}
	}
	if t.EnteredTime != defaultTime {
		_, err = fmt.Fprintf(identityBuilder, "entered time %s\n", t.EnteredTime.Format(timeFormat))
		if err != nil {
			return nil, err
		}
	}
	_, err = fmt.Fprintf(identityBuilder, "amount %s\n", t.Amount)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprintf(identityBuilder, "merchant %s\n", t.Merchant)
	if err != nil {
		return nil, err
	}
	if (t.Committer != User{}) {
		_, err = fmt.Fprintf(identityBuilder, "committer %s\n", t.Committer)
		if err != nil {
			return nil, err
		}
	}
	_, err = fmt.Fprintf(identityBuilder, "comment %s\n", t.Comment)
	if err != nil {
		return nil, err
	}

	return identityBuilder.Bytes(), nil
}
