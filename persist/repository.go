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
	"log/slog"

	"github.com/marstr/envelopes"
)

// BareRepositoryReader indicates that a struct is able to read objects like envelopes.Budget, envelopes.Transaction,
// and branch instances from a repository. However, it does not indicate that the repository has a current working copy.
type BareRepositoryReader interface {
	Loader
	BranchReader
	BranchLister
	TagReader
	TagLister
}

// BareRepositoryWriter indicates that a struct is able to add objects like envelopes.Budget, envelopes.Transaction,
// and branch instances to a repository. However, it does not indicate that the repository has a current working copy.
type BareRepositoryWriter interface {
	Writer
	BranchWriter
	TagWriter
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
			logDiagnostic(ctx, cloneLogger(src, dest), "clone", "options", err)
			return err
		}
	}

	return bareClone(ctx, src, dest, aggregatedOptions)
}

func cloneLogger(src BareRepositoryReader, dest BareRepositoryWriter) *slog.Logger {
	if logger := diagnosticLogger(dest); logger != nil {
		return logger
	}
	return diagnosticLogger(src)
}

func bareClone(ctx context.Context, src BareRepositoryReader, dest BareRepositoryWriter, options cloneOptions) error {
	logger := cloneLogger(src, dest)
	branches, tags, transactions := 0, 0, 0
	logDiagnostic(ctx, logger, "clone", "start", nil, slog.Uint64("max_depth", uint64(options.Depth)))
	finish := func(stage string, err error) error {
		logDiagnostic(ctx, logger, "clone", stage, err,
			slog.Int("branches_written", branches), slog.Int("tags_written", tags),
			slog.Int("transactions_written", transactions))
		return err
	}

	rawBranches, err := src.ListBranches(ctx)
	if err != nil {
		return finish("list_branches", err)
	}

	heads := make([]envelopes.ID, 0)

	for branch := range rawBranches {
		val, err := src.ReadBranch(ctx, branch)
		if err != nil {
			return finish("read_branch", err)
		}
		heads = append(heads, val)
		err = dest.WriteBranch(ctx, branch, val)
		if err != nil {
			return finish("write_branch", err)
		}
		branches++
	}

	rawTags, err := src.ListTags(ctx)
	if err != nil {
		return finish("list_tags", err)
	}

	for tag := range rawTags {
		val, err := src.ReadTag(ctx, tag)
		if err != nil {
			return finish("read_tag", err)
		}
		err = dest.WriteTag(ctx, tag, val)
		if err != nil {
			return finish("write_tag", err)
		}
		tags++
	}

	walker := Walker{
		Loader:   src,
		MaxDepth: options.Depth,
	}

	stage := "load_transaction"
	err = walker.Walk(ctx, func(ctx context.Context, _ envelopes.ID, transaction envelopes.Transaction) error {
		stage = "write_transaction"
		if writeErr := dest.WriteTransaction(ctx, transaction); writeErr != nil {
			return writeErr
		}
		transactions++
		stage = "load_transaction"
		return nil
	}, heads...)
	if err == nil {
		stage = "complete"
	}
	return finish(stage, err)
}

// Commit assigns the currently checked out commit as the parent of the provided transaction, writes that transaction,
// then updates the reference to the currently checkout out branch as appropriate.
func Commit(ctx context.Context, repo RepositoryReaderWriter, transaction envelopes.Transaction, additionalParents ...envelopes.ID) error {
	logger := diagnosticLogger(repo)
	logDiagnostic(ctx, logger, "commit", "start", nil)
	head, err := repo.Current(ctx)
	if err != nil {
		logDiagnostic(ctx, logger, "commit", "current", err)
		return err
	}

	var parent envelopes.ID
	if head != "" {
		parent, err = Resolve(ctx, repo, head)
		if err != nil {
			logDiagnostic(ctx, logger, "commit", "resolve_parent", err)
			return err
		}
	}

	if parent.Equal(envelopes.ID{}) {
		transaction.Parents = []envelopes.ID{}
	} else {
		transaction.Parents = append([]envelopes.ID{parent}, additionalParents...)
	}
	if err != nil {
		return err
	}

	err = repo.WriteTransaction(ctx, transaction)
	if err != nil {
		logDiagnostic(ctx, logger, "commit", "write_transaction", err)
		return err
	}

	id := transaction.ID()
	logDiagnostic(ctx, logger, "commit", "transaction_written", nil,
		slog.Any("object_id", id), slog.Int("parent_count", len(transaction.Parents)))

	_, err = repo.ReadBranch(ctx, string(head))
	if err == nil {
		logDiagnostic(ctx, logger, "commit", "reference_mode", nil, slog.String("mode", "branch"))
		err = repo.WriteBranch(ctx, string(head), id)
		if err != nil {
			logDiagnostic(ctx, logger, "commit", "write_branch", err, slog.Any("object_id", id))
			return err
		}
	} else {
		logDiagnostic(ctx, logger, "commit", "reference_mode", err, slog.String("mode", "detached"))
		err = repo.SetCurrent(ctx, RefSpec(id.String()))
		if err != nil {
			logDiagnostic(ctx, logger, "commit", "set_current", err, slog.Any("object_id", id))
			return err
		}
	}

	logDiagnostic(ctx, logger, "commit", "complete", nil, slog.Any("object_id", id))
	return nil
}
