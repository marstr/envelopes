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
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/marstr/envelopes"
)

const objectsDir = "objects"

// FileSystem allows an easy mechanism for reading and writing Budget related
// objects to and from a hard drive.
type FileSystem struct {
	Root string
}

// LoadCurrent finds the ID of the most recent transaction.
func (fs FileSystem) LoadCurrent(ctx context.Context) (result envelopes.ID, err error) {
	raw, err := ioutil.ReadFile(fs.currentPath())
	if err != nil {
		return
	}

	err = result.UnmarshalText(raw)
	if err != nil {
		return
	}

	return
}

// WriteCurrent makes note of the most recent ID of transaction.
func (fs FileSystem) WriteCurrent(ctx context.Context, current envelopes.Transaction) (err error) {
	writeErr := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(writeErr)
		transformed, err := current.ID().MarshalText()
		if err != nil {
			writeErr <- err
			return
		}

		err = ioutil.WriteFile(fs.currentPath(), transformed, os.ModePerm)
		if err != nil {
			writeErr <- err
			return
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return <-writeErr
	}
}

// Fetch is able to read into memory the marshaled form of a Budget related object.
func (fs FileSystem) Fetch(ctx context.Context, id envelopes.ID) ([]byte, error) {
	return ioutil.ReadFile(fs.path(id))
}

func (fs FileSystem) write(ctx context.Context, target envelopes.IDer) (err error) {
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

// WriteAccounts saves to disk an instance of an `enveloeps.Accounts`.
func (fs FileSystem) WriteAccounts(ctx context.Context, target envelopes.Accounts) error {
	return fs.write(ctx, target)
}

// WriteBudget saves to disk an instance of an `envelopes.Budget`.
func (fs FileSystem) WriteBudget(ctx context.Context, target envelopes.Budget) error {
	return fs.write(ctx, target)
}

// WriteState saves to disk an instance of an `envelopes.State`.
func (fs FileSystem) WriteState(ctx context.Context, target envelopes.State) error {
	return fs.write(ctx, target)
}

// WriteTransaction saves to disk an instance of an `envelopes.Transaction`.
func (fs FileSystem) WriteTransaction(ctx context.Context, target envelopes.Transaction) error {
	return fs.write(ctx, target)
}

func (fs FileSystem) currentPath() string {
	return path.Join(fs.Root, "current.txt")
}

func (fs FileSystem) path(id envelopes.ID) string {
	return path.Join(fs.Root, objectsDir, id.String()+".json")
}
