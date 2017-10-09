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

package distribution

import (
	"github.com/marstr/envelopes"
)

// UpperBound is an inverted `Priority` Distribution. It will first allocate
// funds to a target Budget, then send the rest to an overflow Distributer.
//
// This is a good way to replicate a budget each pay-period. For example, if
// you are paid each two weeks like me, and you try to spend roughly $100 a week
// on groceries, you could set an UpperBound on the Grocery budget $200. This
// way, each time you get a paycheck, you'll automatically allocate whatever it
// takes to get back to your goal. If you underspend on groceries because you
// were travelling, money would automatically get allocated to your other
// priorities. If you overspent on groceries, funds would automatically be
// allocated to back you up.
type UpperBound struct {
	Target        *envelopes.Budget
	SoughtBalance int64
	Overflow      Distributer
}

// LowerBound retrieves funds from a Budget until it reaches a lower bound,
// then proceeds to fetch funds from another Distributer.
type LowerBound UpperBound

// Distribute brings a Budget up to a particular balance, then allocates any
// remaining funds to the overflow. If the amount to be distributed is less
// than or equal to the amount it would take to bring the target budget up to
// the desired balance, the entire amount will be allocated to that budget.
func (ub UpperBound) Distribute(amount int64) (result envelopes.Effect) {
	remaining := ub.SoughtBalance - ub.Target.Balance

	if remaining <= 0 {
		result = ub.Overflow.Distribute(amount)
	} else if split := amount - remaining; split <= 0 {
		result = (*Identity)(ub.Target).Distribute(amount)
	} else {
		result = ub.Overflow.Distribute(split)
		result = result.Add((*Identity)(ub.Target).Distribute(amount - split))
	}
	return
}

// Distribute reduces a budget to a particular balance, then retreives any
// further funds from another Distributer.
func (lb LowerBound) Distribute(amount int64) (result envelopes.Effect) {
	available := lb.Target.Balance - lb.SoughtBalance

	if split := available + amount; split >= 0 {
		result = (*Identity)(lb.Target).Distribute(amount)
	} else {
		result = lb.Overflow.Distribute(split)
		result = result.Add((*Identity)(lb.Target).Distribute(amount - split))
	}
	return
}
