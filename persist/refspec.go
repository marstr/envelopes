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
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"github.com/marstr/envelopes"
)

const (
	// MostRecentTransactionAlias is a string that can be used as a shortcut to represent whichever RefSpec was used to
	// most recently populate the index. It is the same as typing out the most recent branch name or transaction id.
	MostRecentTransactionAlias = "HEAD"
)

type (
	// RefSpec exposes operations on a string that is attempting to specify a particular Transaction ID.
	RefSpec string
)

// ErrNoRefSpec indicates that a particular value was passed as if it could be interpreted as a RefSpec, but it could
// not.
type ErrNoRefSpec string

func (err ErrNoRefSpec) Error() string {
	return fmt.Sprintf("%s is not a valid RefSpec", string(err))
}

var (
	commitPattern = buildRegexpOnce(`^[0-9a-fA-F]{` + strconv.Itoa(2*cap(envelopes.ID{})) + `}$`)
	caretPattern  = buildRegexpOnce(`^(?P<parent>.+)\^$`)
	tildePattern  = buildRegexpOnce(`^(?P<ancestor>.+)~(?P<jumps>\d+)$`)
)

// Resolve interprets a RefSpec that is provided to the envelopes.Transaction ID it is referring to.
func Resolve(ctx context.Context, repo RepositoryReader, subject RefSpec) (envelopes.ID, error) {
	resolved, err := resolveTransactionRefSpec(ctx, repo, subject)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	resolved, err = resolveBranchRefSpec(ctx, repo, subject)
	if err == nil {
		return resolved, err
	}

	resolved, err = resolveMostRecentRefSpec(ctx, repo, subject)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	resolved, err = resolveCaretRefSpec(ctx, repo, subject, Resolve)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	resolved, err = resolveTildeRefSpec(ctx, repo, subject, Resolve)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	return envelopes.ID{}, ErrNoRefSpec(subject)
}

// BareResolve interprets a RefSpec that is provided to the envelopes.Transaction ID it is referring to, but does not support referencing the most recent checked-in transaction.
func BareResolve(ctx context.Context, repo BareRepositoryReader, subject RefSpec) (envelopes.ID, error) {

	resolved, err := resolveTransactionRefSpec(ctx, repo, subject)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	resolved, err = resolveBranchRefSpec(ctx, repo, subject)
	if err == nil {
		return resolved, err
	}

	resolved, err = resolveCaretRefSpec(ctx, repo, subject, BareResolve)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	resolved, err = resolveTildeRefSpec(ctx, repo, subject, BareResolve)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	return envelopes.ID{}, ErrNoRefSpec(subject)
}

// ResolveMany interprets each of the RefSpec's provided, finding the envelopes.Transaction ID they are referring to. The resulting array matches the order of the provided array.
func ResolveMany(ctx context.Context, repo RepositoryReader, refs []RefSpec) ([]envelopes.ID, error) {
	resolved := make([]envelopes.ID, 0, len(refs))
	for _, current := range refs {
		select {
		case <-ctx.Done():
			return []envelopes.ID{}, ctx.Err()
		default:
			// Intentionally Left Blank
		}

		var currentID envelopes.ID
		var err error
		currentID, err = Resolve(ctx, repo, current)
		if err != nil {
			return []envelopes.ID{}, err
		}

		resolved = append(resolved, currentID)
	}

	return resolved, nil
}

// resolveBranchRefSpec find the ID of the Transaction a branch is pointing to.
func resolveBranchRefSpec(ctx context.Context, reader BranchReader, subject RefSpec) (envelopes.ID, error) {
	return reader.ReadBranch(ctx, string(subject))
}

// resolveCaretRefSpec finds the parent ID of the most recent Transaction.
func resolveCaretRefSpec[T Loader](ctx context.Context, repo T, subject RefSpec, recurse func(context.Context, T, RefSpec) (envelopes.ID, error)) (envelopes.ID, error) {
	matches := caretPattern().FindStringSubmatch(string(subject))
	if len(matches) < 2 {
		return envelopes.ID{}, ErrNoRefSpec(subject)
	}

	target, err := recurse(ctx, repo, RefSpec(matches[1]))
	if err != nil {
		return envelopes.ID{}, err
	}

	loaded, err := LoadAncestor(ctx, repo, target, 1)
	if err != nil {
		return envelopes.ID{}, err
	}
	return loaded.ID(), nil
}

// resolveMostRecentRefSpec finds the most recent Transaction ID.
func resolveMostRecentRefSpec(ctx context.Context, repo RepositoryReader, subject RefSpec) (envelopes.ID, error) {
	if subject != MostRecentTransactionAlias {
		return envelopes.ID{}, ErrNoRefSpec(subject)
	}

	currentRefSpec, err := repo.Current(ctx)
	if err != nil {
		return envelopes.ID{}, err
	}

	return BareResolve(ctx, repo, currentRefSpec)
}

// resolveTildeRefSpec scrapes a count of transactions off the end of a RefSpec, resolves the left-hand side, then
// traverses the direct descendents of the specified transactions the number of specified jumps.
func resolveTildeRefSpec[T Loader](ctx context.Context, repo T, subject RefSpec, recurse func(context.Context, T, RefSpec) (envelopes.ID, error)) (envelopes.ID, error) {
	matches := tildePattern().FindStringSubmatch(string(subject))
	if len(matches) < 3 {
		return envelopes.ID{}, ErrNoRefSpec(subject)
	}

	jumps, err := strconv.ParseUint(matches[2], 10, 32)
	if err != nil {
		return envelopes.ID{}, err
	}

	target, err := recurse(ctx, repo, RefSpec(matches[1]))
	if err != nil {
		return envelopes.ID{}, err
	}

	loaded, err := LoadAncestor(ctx, repo, target, uint(jumps))
	if err != nil {
		return envelopes.ID{}, err
	}
	return loaded.ID(), nil
}

// resolveTransactionRefSpec parses a RefSpec which directly specifies a Transaction via text into a binary ID.
func resolveTransactionRefSpec(ctx context.Context, loader Loader, subject RefSpec) (envelopes.ID, error) {
	if !commitPattern().MatchString(string(subject)) {
		return envelopes.ID{}, ErrNoRefSpec(subject)
	}

	var result envelopes.ID
	err := result.UnmarshalText([]byte(subject))
	if err != nil {
		return envelopes.ID{}, err
	}

	// An empty transaction ID (all zeros) is a special case that indicates an uninitialized repository. In this case,
	// no matching object will be found. In all other cases, we want to Resolve the RefSpec only if the object exists.
	if result.Equal(envelopes.ID{}) {
		return result, nil
	}

	var target envelopes.Transaction
	err = loader.LoadTransaction(ctx, result, &target)
	if err != nil {
		return envelopes.ID{}, err
	}

	return result, nil
}

// buildRegexpOnce acts as a getter for regular expressions, lazily compiling patterns exactly once.
func buildRegexpOnce(pattern string) func() *regexp.Regexp {
	var guard sync.Once
	var built *regexp.Regexp
	return func() *regexp.Regexp {
		guard.Do(func() {
			built = regexp.MustCompile(pattern)
		})
		return built
	}
}
