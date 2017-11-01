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

package filesystem

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/marstr/envelopes"
)

type Provider struct {
	RootLocation string
}

func (p Provider) LoadState(id envelopes.ID) (retval envelopes.State, err error) {
	inputLocation := p.targetStatePath(id)
	content, err := ioutil.ReadFile(inputLocation + ".json")
	if err != nil {
		return
	}
	err = json.Unmarshal(content, &retval)
	return
}

func (p Provider) WriteState(subject envelopes.State) error {
	outputLocation := p.targetStatePath(subject.ID())
	outputHandle, err := os.Create(outputLocation)
	if err != nil {
		return err
	}

	content, err := json.Marshal(subject)
	if err != nil {
		return err
	}

	_, err = outputHandle.Write(content)
	return err
}

func (p Provider) targetStatePath(id envelopes.ID) string {
	return filepath.Join(p.RootLocation, "states", fmt.Sprintf("%x", id))
}
