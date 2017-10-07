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
	"fmt"

	env "github.com/marstr/envelopes"
)

// Identity is the most trivial example of a `Distribution` and always returns the amount
// passed to it targeting the underlying `Budget`.
type Identity env.Budget

// Distribute creates a trivial Effect which allocates all funds to a single Budget.
func (id *Identity) Distribute(amount int64) env.Effect {
	return env.Effect{
		(*env.Budget)(id): amount,
	}
}

func (id Identity) String() string {
	return fmt.Sprintf("{%s}", id.Name)
}
