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
	env "github.com/marstr/envelopes"
	"github.com/marstr/envelopes/evaluate"
)

// Branch will evaluate a `condition.Conditioner` then use the appropriate child `Distributer`
type Branch struct {
	Affirmative Distributer
	Negative    Distributer
	Decision    evaluate.Evaluater
}

// Distribute calculates the impact of one of two child `Distributer`s based on how some
// `Conditioner` evaluates.
func (br Branch) Distribute(amount int64) env.Effect {
	var selected Distributer

	if br.Affirmative == br.Negative || br.Decision.Evaluate() {
		selected = br.Affirmative
	} else {
		selected = br.Negative
	}

	return selected.Distribute(amount)
}
