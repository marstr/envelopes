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

	env "github.com/marstr/envelopes"
	"github.com/marstr/envelopes/distribution"
)

func ExamplePercentile_Distribute() {
	grocery := &env.Budget{
		Name:    "Groceries",
		Balance: 9856,
	}

	transport := &env.Budget{
		Name:    "Transit",
		Balance: 4237,
	}

	// create a new Percentile Distribution that allocates 25% of funds to Groceries
	// and 75% of funds to Transport.
	distr, _ := distribution.NewPercentile(map[distribution.Distributer]float64{
		(*distribution.Identity)(grocery):   .25,
		(*distribution.Identity)(transport): .75,
	}, (*distribution.Identity)(grocery))

	// run the created distribution over $10.00
	impacts := distr.Distribute(1000)

	fmt.Println(impacts)
	// Output:
	// ["Transit":$7.50 "Groceries":$2.50]
}

func ExamplePercentile_String() {
	grocery := &env.Budget{
		Name:    "Grocery",
		Balance: 5490,
	}

	transport := &env.Budget{
		Name:    "Transport",
		Balance: 3809,
	}

	distr, _ := distribution.NewPercentile(map[distribution.Distributer]float64{
		(*distribution.Identity)(grocery):   .25,
		(*distribution.Identity)(transport): .75,
	}, (*distribution.Identity)(grocery))

	fmt.Println(distr)
	// Output: {75.00%:{Transport}, 25.00%:{Grocery}}
}
