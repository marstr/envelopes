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

	"github.com/devigned/tab"
	"github.com/marstr/collection"
	"github.com/mitchellh/go-homedir"

	"github.com/marstr/envelopes"
)

const objectsDir = "objects"

// FileSystem allows an easy mechanism for reading and writing Budget related
// objects to and from a hard drive.
type FileSystem struct {
	Root string
}

const fileSystemOperationPrefix = persistOperationPrefix + ".FileSystem"

// Current finds the ID of the most recent transaction.
func (fs FileSystem) Current(ctx context.Context) (result envelopes.ID, err error) {
	var span tab.Spanner
	const operationName = fileSystemOperationPrefix + ".Current"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	p, err := fs.CurrentPath()
	if err != nil {
		span.Logger().Error(err)
		return
	}

	raw, err := ioutil.ReadFile(p)
	if err != nil {
		span.Logger().Error(err)
		return
	}

	trimmed := strings.TrimSpace(string(raw))

	result, err = fs.ReadBranch(ctx, trimmed)
	if err == nil {
		span.Logger().Error(err)
		return
	}

	err = result.UnmarshalText([]byte(trimmed))
	if err != nil {
		span.Logger().Error(err)
		return
	}

	return
}

// WriteCurrent makes note of the most recent ID of transaction. If current.txt currently contains a branch, this
// operation defers to updating the branch file. Should the contents be anything else, the contents of current.txt are
// replaced by the ID of the current Transaction.
func (fs FileSystem) WriteCurrent(ctx context.Context, current *envelopes.Transaction) error {
	var span tab.Spanner
	const operationName = fileSystemOperationPrefix + ".WriteCurrent"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	p, err := fs.CurrentPath()
	if err != nil {
		span.Logger().Error(err)
		return err
	}

	raw, err := ioutil.ReadFile(p)
	if os.IsNotExist(err) {
		const missMessage = "read miss, using default branch (" + DefaultBranch + ")"
		raw = []byte(DefaultBranch)
		span.Logger().Debug(missMessage)
	} else if err != nil {
		span.Logger().Error(err)
		return err
	}

	trimmed := strings.TrimSpace(string(raw))
	_, err = fs.ReadBranch(ctx, trimmed)
	if os.IsNotExist(err) {
		transformed, err := current.ID().MarshalText()
		if err != nil {
			span.Logger().Error(err)
			return err
		}

		return ioutil.WriteFile(p, transformed, os.ModePerm)
	}

	return fs.WriteBranch(ctx, trimmed, current.ID())
}

// Fetch is able to read into memory the marshaled form of a Budget related object.
//
// See Also:
// - FileSystem.Stash
func (fs FileSystem) Fetch(ctx context.Context, id envelopes.ID) ([]byte, error) {
	var span tab.Spanner
	const operationName = fileSystemOperationPrefix + ".Fetch"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	p, err := fs.path(id)
	if err != nil {
		span.Logger().Error(err)
		return nil, err
	}
	span.AddAttributes(tab.StringAttribute("fileLocation", p))

	return ioutil.ReadFile(p)
}

// Stash commits the provided payload to disk at a place that it can retreive again if asked for the ID specified here.
//
// See Also:
// - FileSystem.Fetch
func (fs FileSystem) Stash(ctx context.Context, id envelopes.ID, payload []byte) error {
	var span tab.Spanner
	const operationName = fileSystemOperationPrefix + ".Stash"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	loc, err := fs.path(id)
	if err != nil {
		span.Logger().Error(err)
		return err
	}
	span.AddAttributes(tab.StringAttribute("fileLocation", loc))

	os.MkdirAll(path.Dir(loc), os.ModePerm)
	handle, err := os.Create(loc)
	if err != nil {
		span.Logger().Error(err)
		return err
	}
	defer handle.Close()

	_, err = handle.Write(payload)
	if err != nil {
		span.Logger().Error(err)
		return err
	}
	return nil
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

func (fs FileSystem) branchPath(name string) string {
	return path.Join(fs.Root, "refs", "heads", name)
}

// ReadBranch fetches the ID that a branch is pointing at.
func (fs FileSystem) ReadBranch(ctx context.Context, name string) (retval envelopes.ID, err error) {
	var span tab.Spanner
	const operationName = fileSystemOperationPrefix + ".ReadBranch"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	span.AddAttributes(tab.StringAttribute("branchName", name))

	branchLoc := fs.branchPath(name)
	handle, err := os.Open(branchLoc)
	if err != nil {
		span.Logger().Error(err)
		return
	}
	var contents [2 * cap(retval)]byte
	var n int
	n, err = handle.Read(contents[:])
	span.Logger().Debug(fmt.Sprintf("read %d bytes", n))
	if err != nil {
		span.Logger().Error(err)
		return
	}

	if n != cap(contents) {
		err = fmt.Errorf(
			"%s was not long enough to be a candidate for pointing to a Transaction ID (want: %v got: %v)",
			branchLoc,
			cap(contents),
			n)
		span.Logger().Error(err)
		return
	}

	err = retval.UnmarshalText(contents[:])
	if err != nil {
		span.Logger().Error(err)
		return
	}

	return
}

// WriteBranch sets a branch to be pointing at a particular ID.
func (fs FileSystem) WriteBranch(ctx context.Context, name string, id envelopes.ID) error {
	var span tab.Spanner
	const operationName = fileSystemOperationPrefix + ".WriteBranch"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

	span.AddAttributes(tab.StringAttribute("branchName", name))

	branchLoc := fs.branchPath(name)
	handle, err := os.Create(branchLoc)
	if err != nil {
		return err
	}

	_, err = handle.WriteString(id.String())
	return err
}

func (fs FileSystem) ListBranches(ctx context.Context) (<-chan string, error) {
	var span tab.Spanner
	const operationName = fileSystemOperationPrefix + ".ListBranches"
	ctx, span = tab.StartSpan(ctx, operationName)
	defer span.End()

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
	castResults := make(chan string)
	go func() {
		defer span.End()
		defer close(castResults)

		for entry := range rawResults {
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
