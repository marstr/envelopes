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
	"fmt"
	"math"
	"sort"
)

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
// to the same `Budget`s.
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

func (e Effect) String() string {
	budgets := make([]*Budget, 0, len(e))

	for k := range e {
		budgets = append(budgets, k)
	}

	sort.Slice(budgets, func(i, j int) bool {
		return math.Abs(float64(e[budgets[i]])) >= math.Abs(float64(e[budgets[j]]))
	})

	results := new(bytes.Buffer)

	results.WriteRune('[')

	for i, budg := range budgets {
		if i >= 15 {
			results.WriteString("... ")
			break
		} else if impact := float64(e[budg]) / 100; budg == nil {
			fmt.Fprintf(results, "nil:$%0.2f ", impact)
		} else {
			fmt.Fprintf(results, "%q:$%0.2f ", budg.Name, impact)
		}
	}

	if results.Len() > 1 {
		results.Truncate(results.Len() - 1)
	}

	results.WriteRune(']')
	return results.String()
}
