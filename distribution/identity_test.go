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

package distribution_test

import (
	"fmt"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/distribution"
)

func ExampleIdentity_Distribute() {
	subject := (*distribution.Identity)(&envelopes.Budget{
		Name:    "Grocery",
		Balance: 7843,
	})

	eff := subject.Distribute(1241)
	fmt.Println(eff)
	// Output: ["Grocery":$12.41]
}
