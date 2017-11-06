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
	"time"
)

// Transaction represents one exchange of funds and how it impacted budgets.
type Transaction struct {
	state    ID
	time     time.Time
	amount   int64
	merchant string
	comment  string
	parent   ID
}

// Comment fetches any text supplied by an end user that they would like to associate
// with this transaction.
func (t Transaction) Comment() string {
	return t.comment
}

// SetComment creates a copy of this Transaction that has the updated comment.
func (t Transaction) SetComment(val string) Transaction {
	t.comment = val
	return t
}

// Merchant fetches the name of the party receiving funds.
func (t Transaction) Merchant() string {
	return t.merchant
}

// SetMerchant creates a copy of this Transaction that has the updated Merchant.
func (t Transaction) SetMerchant(val string) Transaction {
	t.merchant = val
	return t
}

// State points to the State of the Budget and all related items
// after the Transaction has taken effect.
func (t Transaction) State() ID {
	return t.state
}

// SetState
func (t Transaction) SetState(val ID) Transaction {
	t.state = val
	return t
}

// Time fetches the moment that this transaction occured.
func (t Transaction) Time() time.Time {
	return t.time
}

func (t Transaction) SetTime(val time.Time) Transaction {
	t.time = val
	return t
}
