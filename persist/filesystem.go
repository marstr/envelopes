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
// and down frequently.
package persist

import (
	"context"
	"github.com/marstr/envelopes"
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

const objectsDir = "objects"

// FileSystem allows an easy mechanism for reading and writing Budget related
// objects to and from a hard drive.
type FileSystem struct {
	Root string
}

// Current finds the ID of the most recent transaction.
func (fs FileSystem) Current(ctx context.Context) (result envelopes.ID, err error) {
	p, err := fs.CurrentPath()
	if err != nil {
		return
	}

	raw, err := ioutil.ReadFile(p)
	if err != nil {
		return
	}

	trimmed := strings.TrimSpace(string(raw))
	raw = []byte(trimmed)

	err = result.UnmarshalText(raw)
	if err != nil {
		return
	}

	return
}

// WriteCurrent makes note of the most recent ID of transaction.
func (fs FileSystem) WriteCurrent(_ context.Context, current *envelopes.Transaction) error {
	transformed, err := current.ID().MarshalText()
	if err != nil {
		return err
	}

	cp, err := fs.CurrentPath()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cp, transformed, os.ModePerm)
}

// Fetch is able to read into memory the marshaled form of a Budget related object.
//
// See Also:
// - FileSystem.Stash
func (fs FileSystem) Fetch(ctx context.Context, id envelopes.ID) ([]byte, error) {
	p, err := fs.path(id)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadFile(p)
}

// Stash commits the provided payload to disk at a place that it can retreive again if asked for the ID specified here.
//
// See Also:
// - FileSystem.Fetch
func (fs FileSystem) Stash(ctx context.Context, id envelopes.ID, payload []byte) error {
	loc, err := fs.path(id)
	if err != nil {
		return err
	}

	os.MkdirAll(path.Dir(loc), os.ModePerm)
	handle, err := os.Create(loc)
	if err != nil {
		return err
	}
	defer handle.Close()

	_, err = handle.Write(payload)
	return err
}

// CurrentPath fetches the name of the file containing the ID to the most up-to-date Transaction.
func (fs FileSystem) CurrentPath() (result string, err error) {
	exp, err := homedir.Expand(fs.Root)
	if err != nil {
		return
	}
	return path.Join(exp, "current.txt"), nil
}

func (fs FileSystem) path(id envelopes.ID) (string, error) {
	exp, err := homedir.Expand(fs.Root)
	if err != nil {
		return "", err
	}
	return path.Join(exp, objectsDir, id.String()+".json"), nil
}
