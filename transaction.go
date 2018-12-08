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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Transaction represents one exchange of funds and how it impacted budgets.
type Transaction struct {
	state    ID
	time     time.Time
	amount   Balance
	merchant string
	comment  string
	parent   ID
}

// ID fetches a SHA1 hash of this object that qill uniquely identify it.
func (t Transaction) ID() (id ID) {
	id, _ = NewID(t)
	return
}

// ParseAmount converts between a string representation of an amount of dollars
// into an int64 number of cents.
func ParseAmount(raw string) (result Balance, err error) {
	raw = strings.TrimPrefix(raw, "$")
	raw = strings.Replace(raw, ",", "", -1)
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return
	}

	if parsed >= 0 {
		result = Balance(parsed*100 + .5)
	} else {
		result = Balance(parsed*100 - .5)
	}
	return
}

// FormatAmount converts an int64 number of cents into a string representation of a number of dollars.
// It is the inverse function of `ParseAmount`
func FormatAmount(amount Balance) (result string) {
	transformed := float64(amount) / 100
	return fmt.Sprintf("$%0.2f", transformed)
}

// Amount is a non-semantic magnitude associated with the transaction.
//
// For most transactions, this amount will match the sum of all diffetences
// in state between each budget in this Transaction's State. However, in the
// case of transfers between two envelopes for the user's own accounting
// purposes, the magnitude of the transfer will always net out as a zero
// impact on the Budgets. Paying off a credit card balance, or just transferring
// money between two envelopes, the magnitude of the transfer is important
// but has zero-sum impact.
func (t Transaction) Amount() Balance {
	return t.amount
}

// WithAmount creates a new Transaction identical in every way to `t` except
// that it will return a different value when `envelopes.Amount()` is called on it.
func (t Transaction) WithAmount(val Balance) Transaction {
	t.amount = val
	return t
}

// Comment fetches any text supplied by an end user that they would like to associate
// with this transaction.
func (t Transaction) Comment() string {
	return t.comment
}

// WithComment creates a copy of this Transaction that has the updated comment.
func (t Transaction) WithComment(val string) Transaction {
	t.comment = val
	return t
}

// Merchant fetches the name of the party receiving funds.
func (t Transaction) Merchant() string {
	return t.merchant
}

// WithMerchant creates a copy of this Transaction that has the updated Merchant.
func (t Transaction) WithMerchant(val string) Transaction {
	t.merchant = val
	return t
}

// Parent fetches the ID of the Transaction that immediately precedes this one logically.
func (t Transaction) Parent() ID {
	return t.parent
}

// WithParent creates a copy of this Transaction with the specifed Parent.
func (t Transaction) WithParent(pid ID) Transaction {
	t.parent = pid
	return t
}

// State points to the State of the Budget and all related items
// after the Transaction has taken effect.
func (t Transaction) State() ID {
	return t.state
}

// WithState creates a copy of this Transaction with the specified State.
func (t Transaction) WithState(val ID) Transaction {
	t.state = val
	return t
}

// Time fetches the moment that this transaction occured.
func (t Transaction) Time() time.Time {
	return t.time
}

// WithTime creates a copy of this Transaction with a given time.
func (t Transaction) WithTime(val time.Time) Transaction {
	t.time = val
	return t
}

// MarshalJSON creates a deterministic JSON object that has the same
// values as `t`.
func (t Transaction) MarshalJSON() ([]byte, error) {
	var err error
	marshaledState, err := json.Marshal(t.State())
	if err != nil {
		return nil, err
	}

	marshaledParent, err := json.Marshal(t.Parent())
	if err != nil {
		return nil, err
	}

	marshaledComment, err := json.Marshal(t.Comment())
	if err != nil {
		return nil, err
	}

	marshaledTime, err := json.Marshal(t.Time())
	if err != nil {
		return nil, err
	}

	marshaledMerchant, err := json.Marshal(t.Merchant())
	if err != nil {
		return nil, err
	}

	builder := new(bytes.Buffer)

	// It is important that fixed width items come first. This allows for
	// persistance strategies that use JSON written to disk to not fully
	// unmarshal items to get the IDs of child items such as the parent and
	// state associated with this transaction. (They can do this by reading into
	// a []byte and then index on the known positions of state and parent IDs.)
	builder.WriteRune('{')

	builder.WriteString(`"state":`)
	builder.Write(marshaledState)
	builder.WriteRune(',')

	builder.WriteString(`"parent":`)
	builder.Write(marshaledParent)
	builder.WriteRune(',')

	builder.WriteString(`"amount":`)
	fmt.Fprint(builder, t.Amount())
	builder.WriteRune(',')

	builder.WriteString(`"comment":`)
	builder.Write(marshaledComment)
	builder.WriteRune(',')

	builder.WriteString(`"time":`)
	builder.Write(marshaledTime)
	builder.WriteRune(',')

	builder.WriteString(`"merchant":`)
	builder.Write(marshaledMerchant)

	builder.WriteRune('}')

	return builder.Bytes(), nil
}

func (t *Transaction) UnmarshalJSON(content []byte) (err error) {
	var intermediate map[string]json.RawMessage

	err = json.Unmarshal(content, &intermediate)
	if err != nil {
		return
	}

	err = json.Unmarshal(intermediate["comment"], &t.comment)
	if err != nil {
		return
	}

	err = json.Unmarshal(intermediate["time"], &t.time)
	if err != nil {
		return
	}

	err = json.Unmarshal(intermediate["parent"], &t.parent)
	if err != nil {
		return
	}

	err = json.Unmarshal(intermediate["state"], &t.state)
	if err != nil {
		return
	}

	err = json.Unmarshal(intermediate["amount"], &t.amount)
	if err != nil {
		return
	}

	err = json.Unmarshal(intermediate["merchant"], &t.merchant)
	return
}
