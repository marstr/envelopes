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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/marstr/collection"
	"github.com/mitchellh/go-homedir"

	"github.com/marstr/envelopes"
)

const objectsDir = "objects"

// FileSystem allows an easy mechanism for reading and writing Budget related
// objects to and from a hard drive.
type FileSystem struct {
	Root              string
	CreatePermissions os.FileMode
}

func (fs FileSystem) getCreatePermissions() os.FileMode {
	if fs.CreatePermissions == 0 {
		return 0770
	}
	return fs.CreatePermissions
}

func (fs FileSystem) Current(_ context.Context) (result RefSpec, err error) {
	p, err := fs.currentPath()
	if err != nil {
		return
	}

	raw, err := ioutil.ReadFile(p)
	if err != nil {
		return
	}

	trimmed := strings.TrimSpace(string(raw))
	result = RefSpec(trimmed)
	return
}

// WriteCurrent makes note of the most recent ID of transaction. If current.txt currently contains a branch, this
// operation defers to updating the branch file. Should the contents be anything else, the contents of current.txt are
// replaced by the ID of the current Transaction.
func (fs FileSystem) WriteCurrent(ctx context.Context, current envelopes.Transaction) error {
	p, err := fs.currentPath()
	if err != nil {
		return err
	}

	raw, err := ioutil.ReadFile(p)
	if os.IsNotExist(err) {
		raw = []byte(DefaultBranch)
	} else if err != nil {
		return err
	}

	trimmed := strings.TrimSpace(string(raw))
	_, err = fs.ReadBranch(ctx, trimmed)
	if os.IsNotExist(err) {
		transformed, err := current.ID().MarshalText()
		if err != nil {
			return err
		}

		return ioutil.WriteFile(p, transformed, fs.getCreatePermissions())
	}

	return fs.WriteBranch(ctx, trimmed, current.ID())
}

// SetCurrent replaces the current pointer to the most recent Transaction with a given RefSpec. For instance, this
// should be used to change which branch is currently checked-out.
func (fs FileSystem) SetCurrent(ctx context.Context, current RefSpec) error {
	p, err := fs.currentPath()
	if err != nil {
		return err
	}

	if _, err = fs.ReadBranch(ctx, string(current)); err == nil {
		return ioutil.WriteFile(p, []byte(current), fs.getCreatePermissions())
	} else if os.IsNotExist(err) {
		resolver := RefSpecResolver{
			Loader:   DefaultLoader{Fetcher: fs},
			Brancher: fs,
			Fetcher:  fs,
		}

		id, err := resolver.Resolve(ctx, current)
		if err != nil {
			return err
		}

		marshaled, err := id.MarshalText()
		if err != nil {
			return err
		}
		return ioutil.WriteFile(p, marshaled, fs.getCreatePermissions())
	}

	return err
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

	err = os.MkdirAll(path.Dir(loc), fs.getCreatePermissions())
	if err != nil {
		return err
	}

	handle, err := os.Create(loc)
	if err != nil {
		return err
	}
	defer handle.Close()

	_, err = handle.Write(payload)
	return err
}

// currentPath fetches the name of the file containing the ID to the most up-to-date Transaction.
func (fs FileSystem) currentPath() (result string, err error) {
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

func (fs FileSystem) branchPath(name string) string {
	return path.Join(fs.Root, "refs", "heads", name)
}

// ReadBranch fetches the ID that a branch is pointing at.
func (fs FileSystem) ReadBranch(_ context.Context, name string) (retval envelopes.ID, err error) {
	branchLoc := fs.branchPath(name)
	handle, err := os.Open(branchLoc)
	if err != nil {
		return
	}
	var contents [2 * cap(retval)]byte
	var n int
	n, err = handle.Read(contents[:])
	if err != nil {
		return
	}

	if expected := cap(contents); n != expected {
		err = fmt.Errorf(
			"%s was not long enough to be a candidate for pointing to a Transaction ID (want: %v got: %v)",
			branchLoc,
			expected,
			n)
		return
	}

	err = retval.UnmarshalText(contents[:])
	return
}

// WriteBranch sets a branch to be pointing at a particular ID.
func (fs FileSystem) WriteBranch(_ context.Context, name string, id envelopes.ID) error {
	branchLoc := fs.branchPath(name)

	err := os.MkdirAll(filepath.Dir(branchLoc), fs.getCreatePermissions())
	if err != nil {
		return err
	}

	handle, err := os.Create(branchLoc)
	if err != nil {
		return err
	}
	defer handle.Close()

	_, err = handle.WriteString(id.String())
	return err
}

func (fs FileSystem) ListBranches(ctx context.Context) (<-chan string, error) {
	absRoot, err := filepath.Abs(path.Dir(fs.branchPath("any_branch_name")))
	if err != nil {
		return nil, err
	}

	dir := collection.Directory{
		Location: absRoot,
		Options:  collection.DirectoryOptionsExcludeDirectories | collection.DirectoryOptionsRecursive,
	}

	rawResults := dir.Enumerate(ctx.Done())

	prefix := absRoot + "/"
	prefix = strings.Replace(prefix, "\\", "/", -1)
	castResults := make(chan string)
	go func() {
		defer close(castResults)

		for entry := range rawResults {
			entry = strings.Replace(entry.(string), "\\", "/", -1)
			trimmed := strings.TrimPrefix(entry.(string), prefix)
			select {
			case <-ctx.Done():
				return
			case castResults <- trimmed:
				// Intentionally Left Blank
			}
		}
	}()

	return castResults, nil
}

// Clone uses an arbitrary Loader to retrieve all transactions that are ancestors of a specified Transaction. As it
// acquires each Transaction, it writes it to disk.
func (fs FileSystem) Clone(ctx context.Context, walkStart envelopes.ID, branchName string, loader Loader) error {
	err := fs.copyTransactions(ctx, walkStart, loader)
	if err != nil {
		return err
	}

	err = fs.WriteBranch(ctx, branchName, walkStart)
	if err != nil {
		return err
	}

	localLoader := DefaultLoader{Fetcher: fs}
	var head envelopes.Transaction
	err = localLoader.Load(ctx, walkStart, &head)
	if err != nil {
		return err
	}

	return fs.SetCurrent(ctx, RefSpec(branchName))
}

func (fs FileSystem) copyTransactions(ctx context.Context, walkStart envelopes.ID, loader Loader) error {
	next := walkStart

	writer := DefaultWriter{
		Stasher: fs,
	}

	var current envelopes.Transaction
	for {
		err := loader.Load(ctx, next, &current)
		if err != nil {
			return err
		}

		err = writer.Write(ctx, current)
		if err != nil {
			return err
		}

		if current.Parent.Equal(envelopes.ID{}) {
			break
		}
		next = current.Parent
	}

	return nil
}
