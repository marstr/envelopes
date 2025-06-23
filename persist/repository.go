/*
 * Copyright {YEAR} Martin Strobel
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package persist

import (
	"context"

	"github.com/marstr/envelopes"
)

// BareRepositoryReader indicates that a struct is able to read objects like envelopes.Budget, envelopes.Transaction,
// and branch instances from a repository. However, it does not indicate that the repository has a current working copy.
type BareRepositoryReader interface {
	Loader
	BranchReader
	BranchLister
}

// BareRepositoryWriter indicates that a struct is able to add objects like envelopes.Budget, envelopes.Transaction,
// and branch instances to a repository. However, it does not indicate that the repository has a current working copy.
type BareRepositoryWriter interface {
	Writer
	BranchWriter
}

// BareRepositoryReaderWriter indicates that a struct has the ability to both read and write
type BareRepositoryReaderWriter interface {
	BareRepositoryReader
	BareRepositoryWriter
}

// RepositoryReader indicates that a structs has all the functionality of a BareRepositoryReader, plus a CurrentReader.
type RepositoryReader interface {
	BareRepositoryReader
	CurrentReader
}

// RepositoryWriter has all the functionality of a BareRepositoryWriter, plus the ability to update Current.
type RepositoryWriter interface {
	BareRepositoryWriter
	CurrentWriter
}

// RepositoryReaderWriter has all the functionality of a RepositoryReader and RepositoryWriter.
type RepositoryReaderWriter interface {
	RepositoryReader
	RepositoryWriter
}

type cloneOptions struct {
	Depth uint
}

type CloneOption func(options *cloneOptions) error

// CloneDepth limits the number of transactions that will be copied by BareClone by indicating the number of generations
// it should traverse.
func CloneDepth(depth uint) CloneOption {
	return func(options *cloneOptions) error {
		options.Depth = depth
		return nil
	}
}

// BareClone retrieves budget objects from `src` and duplicates them at `dest`.
func BareClone(ctx context.Context, src BareRepositoryReader, dest BareRepositoryWriter, options ...CloneOption) error {
	aggregatedOptions := cloneOptions{}
	for _, option := range options {
		if err := option(&aggregatedOptions); err != nil {
			return err
		}
	}

	return bareClone(ctx, src, dest, aggregatedOptions)
}

func bareClone(ctx context.Context, src BareRepositoryReader, dest BareRepositoryWriter, options cloneOptions) error {
	rawBranches, err := src.ListBranches(ctx)
	if err != nil {
		return err
	}

	heads := make([]envelopes.ID, 0)

	for branch := range rawBranches {
		val, err := src.ReadBranch(ctx, branch)
		if err != nil {
			return err
		}
		heads = append(heads, val)
		err = dest.WriteBranch(ctx, branch, val)
		if err != nil {
			return err
		}
	}

	walker := Walker{
		Loader:   src,
		MaxDepth: options.Depth,
	}

	return walker.Walk(ctx, func(ctx context.Context, _ envelopes.ID, transaction envelopes.Transaction) error {
		return dest.WriteTransaction(ctx, transaction)
	}, heads...)
}

// Commit assigns the currently checked out commit as the parent of the provided transaction, writes that transaction,
// then updates the reference to the currently checkout out branch as appropriate.
func Commit(ctx context.Context, repo RepositoryReaderWriter, transaction envelopes.Transaction) error {
	head, err := repo.Current(ctx)
	if err != nil {
		return err
	}

	var parent envelopes.ID
	if head != "" {
		parent, err = Resolve(ctx, repo, head)
		if err != nil {
			return err
		}
	}

	if parent.Equal(envelopes.ID{}) {
		transaction.Parents = []envelopes.ID{}
	} else {
		transaction.Parents = []envelopes.ID{parent}
	}

	err = repo.WriteTransaction(ctx, transaction)
	if err != nil {
		return err
	}

	_, err = repo.ReadBranch(ctx, string(head))
	if err == nil {
		err = repo.WriteBranch(ctx, string(head), transaction.ID())
		if err != nil {
			return err
		}
	} else {
		err = repo.SetCurrent(ctx, RefSpec(transaction.ID().String()))
		if err != nil {
			return err
		}
	}

	return nil
}
