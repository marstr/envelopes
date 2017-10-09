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
	time.Time
	Account
	Effect
	Amount   int64
	Merchant string
}

func (t Transaction) Apply() {
	for budg, adj := range t.Effect {
		budg.Balance += adj
	}
}

func (t Transaction) Undo() {
	negated := t.Effect.Negate()
	for budg, adj := range negated {
		budg.Balance += adj
	}
}
