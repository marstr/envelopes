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

// Package persist defines the basic requirements that the object model expects
// in order to save and load state. The object model expects to be spun up
// and down frequently
package persist

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/marstr/envelopes"
)

const objectsDir = "objects"

// FileSystem allows an easy mechanism for reading and writing Budget related
// objects to and from a hard drive.
type FileSystem struct {
	Root string
}

// Fetch is able to read into memory the marshaled form of a Budget related object.
func (fs FileSystem) Fetch(id envelopes.ID) ([]byte, error) {
	return ioutil.ReadFile(fs.path(id))
}

func (fs FileSystem) write(target envelopes.IDer) (err error) {
	loc := fs.path(target.ID())
	os.MkdirAll(path.Dir(loc), os.ModePerm)
	handle, err := os.Create(loc)
	if err != nil {
		return
	}
	defer handle.Close()

	marshaled, err := json.Marshal(target)
	if err != nil {
		return
	}

	_, err = handle.Write(marshaled)
	return
}

// WriteBudget saves to disk an instance of an `envelopes.Budget`.
func (fs FileSystem) WriteBudget(target envelopes.Budget) error {
	return fs.write(target)
}

// WriteState saves to disk an instance of an `envelopes.State`.
func (fs FileSystem) WriteState(target envelopes.State) error {
	return fs.write(target)
}

// WriteTransaction saves to disk an instance of an `envelopes.Transaction`.
func (fs FileSystem) WriteTransaction(target envelopes.Transaction) error {
	return fs.write(target)
}

func (fs FileSystem) path(id envelopes.ID) string {
	return filepath.Join(fs.Root, objectsDir, id.String()+".json")
}
