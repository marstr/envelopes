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
	"errors"

	"github.com/marstr/envelopes"
)

type MockRepository struct {
	branches map[string]envelopes.ID
	current  RefSpec
	Cache
}

func NewMockRepository(branchCapacity uint, transactionCapacity uint) *MockRepository {
	return &MockRepository{
		branches: make(map[string]envelopes.ID, branchCapacity),
		current:  "",
		Cache:    *NewCache(transactionCapacity),
	}
}

func (mb MockRepository) ReadBranch(_ context.Context, name string) (envelopes.ID, error) {
	retval, ok := mb.branches[name]
	if !ok {
		return envelopes.ID{}, errors.New("no such branch")
	}
	return retval, nil
}

func (mb MockRepository) WriteBranch(_ context.Context, name string, id envelopes.ID) error {
	mb.branches[name] = id
	return nil
}

func (mb MockRepository) ListBranches(ctx context.Context) (<-chan string, error) {
	retval := make(chan string)

	func() {
		defer close(retval)
		for k := range mb.branches {
			select {
			case <-ctx.Done():
				return
			case retval <- k:
				// Intentionally Left Blank
			}
		}
	}()

	return retval, nil
}

func (mb MockRepository) Current(_ context.Context) (RefSpec, error) {
	return mb.current, nil
}

func (mb *MockRepository) WriteCurrent(_ context.Context, current envelopes.Transaction) error {
	if _, ok := mb.branches[string(mb.current)]; ok {
		mb.branches[string(mb.current)] = current.ID()
	} else {
		mb.current = RefSpec(current.ID().String())
	}
	return nil
}

func (mb *MockRepository) SetCurrent(_ context.Context, current RefSpec) error {
	mb.current = current
	return nil
}
