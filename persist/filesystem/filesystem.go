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
package filesystem

import (
	"context"
	"fmt"
	"github.com/marstr/collection/v2"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"

	"github.com/mitchellh/go-homedir"
)

const objectsDir = "objects"

// FileSystem allows an easy mechanism for reading and writing raw Budget related objects to and from a hard drive.
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

// Current fetches the RefSpec that was most recently used to populate the index.
func (fs FileSystem) Current(_ context.Context) (result persist.RefSpec, err error) {
	p, err := fs.currentPath()
	if err != nil {
		return
	}

	raw, err := ioutil.ReadFile(p)
	if err != nil {
		return
	}

	trimmed := strings.TrimSpace(string(raw))
	result = persist.RefSpec(trimmed)
	return
}

// SetCurrent replaces the current pointer to the most recent Transaction with a given RefSpec. For instance, this
// should be used to change which branch is currently checked-out.
func (fs FileSystem) SetCurrent(_ context.Context, current persist.RefSpec) error {
	p, err := fs.currentPath()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(p, []byte(current), fs.getCreatePermissions())
}

// Fetch is able to read into memory the marshaled form of a Budget related object.
//
// See Also:
// - FileSystem.Stash
func (fs FileSystem) Fetch(_ context.Context, id envelopes.ID) ([]byte, error) {
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
func (fs FileSystem) Stash(_ context.Context, id envelopes.ID, payload []byte) error {
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

// ListBranches fetches the distinct names of the branches that exist in a repository.
func (fs FileSystem) ListBranches(ctx context.Context) (<-chan string, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	absRoot, err := filepath.Abs(path.Dir(fs.branchPath("any_branch_name")))
	if err != nil {
		return nil, err
	}

	dir := collection.Directory{
		Location: absRoot,
		Options:  collection.DirectoryOptionsExcludeDirectories | collection.DirectoryOptionsRecursive,
	}

	rawResults := dir.Enumerate(ctx)

	prefix := absRoot + "/"
	prefix = strings.Replace(prefix, "\\", "/", -1)
	castResults := make(chan string)
	go func() {
		defer close(castResults)

		for entry := range rawResults {
			entry = strings.Replace(entry, "\\", "/", -1)
			trimmed := strings.TrimPrefix(entry, prefix)
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
