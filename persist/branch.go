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

package persist

import (
	"context"

	"github.com/marstr/envelopes"
)

const (
	// DefaultBranch is the name of the ref that will be used to to create a repository should no other branch name be
	// specified.
	DefaultBranch = "master"
)

// BranchReader indicates that a type is capable of discovering the envelopes.Transaction that a branch points at.
type BranchReader interface {
	ReadBranch(ctx context.Context, name string) (envelopes.ID, error)
}

// BranchWriter indicates that a type is capable of update the envelopes.Transaction that a branch points at.
type BranchWriter interface {
	WriteBranch(ctx context.Context, name string, id envelopes.ID) error
}

// BranchReaderWriter indicates that a types has both the capabilities of a BranchReader and BranchWriter.
type BranchReaderWriter interface {
	BranchReader
	BranchWriter
}

// BranchLister are able to find all branches is a repository.
type BranchLister interface {
	ListBranches(ctx context.Context) (<-chan string, error)
}
