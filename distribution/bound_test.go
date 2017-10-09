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

func ExampleUpperBound_Distribute() {
	grocery, transit := &envelopes.Budget{
		Name:    "Groceries",
		Balance: 8420,
	}, &envelopes.Budget{
		Name:    "Transit",
		Balance: 2278,
	}

	subject := distribution.UpperBound{
		Target:        grocery,
		SoughtBalance: 10000,
		Overflow:      (*distribution.Identity)(transit),
	}

	fmt.Println(subject.Distribute(4000))
	fmt.Println(subject.Distribute(1000))
	// Output:
	// ["Transit":$24.20 "Groceries":$15.80]
	// ["Groceries":$10.00]
}
