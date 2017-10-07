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
	"bytes"
	"errors"
	"fmt"
	"sort"

	env "github.com/marstr/envelopes"
)

// Percentile will proportionaly distribute funds over a set number
type Percentile struct {
	targets  map[Distributer]float64
	overflow Distributer
}

// NewPercentile validates and instantiates
func NewPercentile(targets map[Distributer]float64, overflow Distributer) (created Percentile, err error) {
	if overflow == nil {
		err = errors.New("percentile distributions must have a valid overflow recipient")
	} else {
		created.overflow = overflow
	}

	// TODO, should I clone this for safety?
	created.targets = targets

	return
}

// Distribute determines what impact would be felt if proportionaly applying the specified
// amount.
func (p Percentile) Distribute(amount int64) (result env.Effect) {
	remaining := amount
	result = make(env.Effect)

	for recipient, adjustment := range p.targets {
		realized := int64(adjustment * float64(amount))
		remaining -= realized
		result = result.Add(recipient.Distribute(realized))
	}

	result.Add(p.overflow.Distribute(remaining))
	return
}

func (p Percentile) String() string {
	builder := bytes.NewBufferString("{")

	contents := []Distributer{}
	for recipient := range p.targets {
		contents = append(contents, recipient)
	}
	sort.Slice(contents, func(i, j int) bool {
		return p.targets[contents[i]] >= p.targets[contents[j]]
	})

	any := false
	for _, entry := range contents {
		any = true
		fmt.Fprintf(builder, "%0.2f%%:%v, ", 100*p.targets[entry], entry)
	}

	if any {
		builder.Truncate(builder.Len() - 2) // Remove final ", "
	}
	fmt.Fprint(builder, "}")
	return builder.String()
}
