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
	"strings"
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
	RecordID    BankRecordID
	Parents     []ID
	Reverts     []ID
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

// Equal determines whether two instances of Transaction share identical values.
func (t Transaction) Equal(other Transaction) bool {
	if t.State == nil && other.State != nil {
		return false
	} else if t.State == nil && other.State == nil {
		// Intentionally Left Blank
	} else if !t.State.Equal(*other.State) {
		return false
	}

	if !t.ActualTime.Equal(other.ActualTime) {
		return false
	}

	if !t.PostedTime.Equal(other.PostedTime) {
		return false
	}

	if !t.EnteredTime.Equal(other.EnteredTime) {
		return false
	}

	if !t.Amount.Equal(other.Amount) {
		return false
	}

	if t.Merchant != other.Merchant {
		return false
	}

	if !t.Committer.Equal(other.Committer) {
		return false
	}

	if t.Comment != other.Comment {
		return false
	}

	if t.RecordID != other.RecordID {
		return false
	}

	if len(t.Reverts) != len(other.Reverts) {
		return false
	}

	for i := range t.Reverts {
		if !t.Reverts[i].Equal(other.Reverts[i]) {
			return false
		}
	}

	if len(t.Parents) != len(other.Parents) {
		return false
	}

	for i := range t.Parents {
		if !t.Parents[i].Equal(other.Parents[i]) {
			return false
		}
	}

	return true
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

	strParents := make([]string, 0, len(t.Parents))
	for i := range t.Parents {
		strParents = append(strParents, t.Parents[i].String())
	}
	joinedParents := strings.Join(strParents, ",")
	_, err = fmt.Fprintf(identityBuilder, "parents %s\n", joinedParents)
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
	if t.RecordID != "" {
		_, err = fmt.Fprintf(identityBuilder, "record %s\n", t.RecordID)
		if err != nil {
			return nil, err
		}
	}
	if len(t.Reverts) > 0 {
		strRevs := make([]string, len(t.Reverts))
		for i := range t.Reverts {
			strRevs[i] = t.Reverts[i].String()
		}
		_, err = fmt.Fprintf(identityBuilder, "reverts %s\n", strings.Join(strRevs, ","))
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
