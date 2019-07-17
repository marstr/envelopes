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

	"github.com/devigned/tab"

	"github.com/marstr/envelopes"
)

const (
	MostRecentTransactionAlias = "HEAD"
)

const refSpecResolverOperationPrefix = persistOperationPrefix + ".RefSpecResolver"

type (
	// RefSpec exposes operations on a string that is attempting to specify a particular Transaction ID.
	RefSpec string

	// RefSpecResolver provides a mechanism for converting a RefSpec into a Transaction ID.
	RefSpecResolver struct {
		Loader
		Brancher
		Fetcher
	}
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

// Resolve interprets a RefSpec that is provided to the Transaction ID it is referring to.
func (resolver RefSpecResolver) Resolve(ctx context.Context, subject RefSpec) (envelopes.ID, error) {
	var span tab.Spanner
	const operationName = refSpecResolverOperationPrefix + ".Resolve"
	ctx, span = tab.StartSpan(ctx, operationName)
	span.AddAttributes(tab.StringAttribute("subject", string(subject)))
	defer span.End()

	resolved, err := resolver.resolveTransactionRefSpec(ctx, subject)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	resolved, err = resolver.resolveBranchRefSpec(ctx, subject)
	if err == nil {
		return resolved, err
	}

	resolved, err = resolver.resolveMostRecentRefSpec(ctx, subject)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	resolved, err = resolver.resolveCaretRefSpec(ctx, subject)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	resolved, err = resolver.resolveTildeRefSpec(ctx, subject)
	if _, ok := err.(ErrNoRefSpec); !ok {
		return resolved, err
	}

	err = ErrNoRefSpec(subject)
	span.Logger().Error(err)
	return envelopes.ID{}, err
}

// resolveBranchRefSpec find the ID of the Transaction a branch is pointing to.
func (resolver RefSpecResolver) resolveBranchRefSpec(ctx context.Context, subject RefSpec) (envelopes.ID, error) {
	var span tab.Spanner
	const operationName = refSpecResolverOperationPrefix + ".resolveBranchRefSpec"
	ctx, span = tab.StartSpan(ctx, operationName)
	span.AddAttributes(tab.StringAttribute("subject", string(subject)))
	defer span.End()

	return resolver.ReadBranch(ctx, string(subject))
}

// resolveCaretRefSpec finds the parent ID of the most recent Transaction.
func (resolver RefSpecResolver) resolveCaretRefSpec(ctx context.Context, subject RefSpec) (envelopes.ID, error) {
	var span tab.Spanner
	const operationName = refSpecResolverOperationPrefix + ".resolveCaretRefSpec"
	ctx, span = tab.StartSpan(ctx, operationName)
	span.AddAttributes(tab.StringAttribute("subject", string(subject)))
	defer span.End()

	matches := caretPattern().FindStringSubmatch(string(subject))
	if len(matches) < 2 {
		err := ErrNoRefSpec(subject)
		span.Logger().Error(err)
		return envelopes.ID{}, err
	}

	target, err := resolver.Resolve(ctx, RefSpec(matches[1]))
	if err != nil {
		span.Logger().Error(err)
		return envelopes.ID{}, err
	}

	loaded, err := LoadAncestor(ctx, resolver, target, 1)
	if err != nil {
		span.Logger().Error(err)
		return envelopes.ID{}, err
	}
	return loaded.ID(), nil
}

// resolveMostRecentRefSpec finds the most recent Transaction ID.
func (resolver RefSpecResolver) resolveMostRecentRefSpec(ctx context.Context, subject RefSpec) (envelopes.ID, error) {
	var span tab.Spanner
	const operationName = refSpecResolverOperationPrefix + ".resolveMostRecentRefSpec"
	ctx, span = tab.StartSpan(ctx, operationName)
	span.AddAttributes(tab.StringAttribute("subject", string(subject)))
	defer span.End()

	if subject != MostRecentTransactionAlias {
		err := ErrNoRefSpec(subject)
		span.Logger().Error(err)
		return envelopes.ID{}, err
	}

	return resolver.Current(ctx)
}

// resolveTildeRefSpec scrapes a count of transactions off the end of a RefSpec, resolves the left-hand side, then
// traverses the direct descendents of the specified transactions the number of specified jumps.
func (resolver RefSpecResolver) resolveTildeRefSpec(ctx context.Context, subject RefSpec) (envelopes.ID, error) {
	var span tab.Spanner
	const operationName = refSpecResolverOperationPrefix + ".resolveTildeRefSpec"
	ctx, span = tab.StartSpan(ctx, operationName)
	span.AddAttributes(tab.StringAttribute("subject", string(subject)))
	defer span.End()

	matches := tildePattern().FindStringSubmatch(string(subject))
	if len(matches) < 3 {
		err := ErrNoRefSpec(subject)
		span.Logger().Error(err)
		return envelopes.ID{}, err
	}

	jumps, err := strconv.ParseUint(matches[2], 10, 32)
	if err != nil {
		span.Logger().Error(err)
		return envelopes.ID{}, err
	}

	target, err := resolver.Resolve(ctx, RefSpec(matches[1]))
	if err != nil {
		span.Logger().Error(err)
		return envelopes.ID{}, err
	}

	loaded, err := LoadAncestor(ctx, resolver, target, uint(jumps))
	if err != nil {
		span.Logger().Error(err)
		return envelopes.ID{}, err
	}
	return loaded.ID(), nil
}

// resolveTransactionRefSpec parses a RefSpec which directly specifies a Transaction via text into a binary ID.
func (resolver RefSpecResolver) resolveTransactionRefSpec(ctx context.Context, subject RefSpec) (envelopes.ID, error) {
	var span tab.Spanner
	const operationName = refSpecResolverOperationPrefix + ".resolveTransactionRefSpec"
	ctx, span = tab.StartSpan(ctx, operationName)
	span.AddAttributes(tab.StringAttribute("subject", string(subject)))
	defer span.End()

	if !commitPattern().MatchString(string(subject)) {
		err := ErrNoRefSpec(subject)
		span.Logger().Error(err)
		return envelopes.ID{}, err
	}

	var result envelopes.ID
	err := result.UnmarshalText([]byte(subject))
	if err != nil {
		span.Logger().Error(err)
		return envelopes.ID{}, err
	}

	var target envelopes.Transaction
	err = resolver.Load(ctx, result, &target)
	if err != nil {
		span.Logger().Error(err)
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
