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
	"io"

	"github.com/marstr/envelopes"
)

// Fetcher can grab the marshaled form of an Object given an ID.
type Fetcher interface {
	Fetch(context.Context, envelopes.ID) ([]byte, error)
	// FetchReadCloser returns a ReadCloser for the marshaled form of an Object.
	// Callers must close the returned ReadCloser when they are done reading.
	FetchReadCloser(context.Context, envelopes.ID) (io.ReadCloser, error)
}
