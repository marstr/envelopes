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

// Writer defines a contract that allows an object to express that it knows how to persist
// an object so that it can be recalled using an instance of an object that satisfies `persist.Fetch`.
type Writer interface {
	// Write persists an object in durable storage where it can be retrieved later.
	Write(ctx context.Context, subject envelopes.IDer) error
}
