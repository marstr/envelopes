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

import env "github.com/marstr/envelopes"

// Priority places the first `Magnitude` of funds in `Target` and puts any
// remaining funds in `Overflow`.
type Priority struct {
	Target    Distributer
	Magnitude uint64
	Overflow  Distributer
}

// Distribute allocations funds first to a `Target` `Priotity` before
// allocating funds to an overflow amount.
func (p Priority) Distribute(amount int64) (result env.Effect) {
	var mag uint64
	if amount >= 0 {
		mag = uint64(amount)
	} else {
		mag = uint64(-1 * amount)
	}

	if mag <= p.Magnitude {
		result = p.Target.Distribute(amount)
	} else if amount >= 0 {
		result = p.Target.Distribute(int64(p.Magnitude))
		result = result.Add(p.Overflow.Distribute(amount - int64(p.Magnitude)))
	} else {
		result = p.Target.Distribute(-1 * int64(p.Magnitude))
		result = result.Add(p.Overflow.Distribute(amount + int64(p.Magnitude)))
	}
	return
}
