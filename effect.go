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

// Effect is a collection of amounts that Budgets will be impacted.
type Effect map[*Budget]int64

// Add creates a new `Effect` that has the cumulative impacts of two `Effect` instances.
func (e Effect) Add(other Effect) (result Effect) {
	result = make(map[*Budget]int64)

	for budg, adjustment := range e {
		result[budg] = adjustment
	}

	for budg, adjustment := range other {
		if _, ok := e[budg]; ok {
			result[budg] += adjustment
		} else {
			result[budg] = adjustment
		}
	}

	result.Truncate()

	return
}

// Truncate removes any adjustments to a `Budget` that has the value 0.
func (e Effect) Truncate() {
	for budg, adj := range e {
		if adj == 0 {
			delete(e, budg)
		}
	}
}

// Equal scans two `Effect`s and returns whether or not they contain the same adjustments
// to the same `Budget`s. Zero impact adjustments are `Truncate`d before evaluation.
func (e Effect) Equal(other Effect) bool {
	e.Truncate()
	other.Truncate()

	if len(e) != len(other) {
		return false
	}

	for budget, adjustment := range e {
		if otherAdj, ok := e[budget]; !(ok && otherAdj == adjustment) {
			return false
		}
	}
	return true
}

// Negate creates an `Effect` with the opposite impact of this one.
func (e Effect) Negate() (result Effect) {
	result = make(Effect)

	for budg, adj := range e {
		result[budg] = -adj
	}

	return
}
