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
	"encoding/json"

	"github.com/marstr/envelopes"
)

// Loader can instantiate core envelopes objects given just an ID.
type Loader interface {
	Load(ctx context.Context, id envelopes.ID, destination envelopes.IDer) error
}

// DefaultLoader wraps a Fetcher and does just the unmarshaling portion.
type DefaultLoader struct {
	Fetcher
}

func (dl DefaultLoader) Load(ctx context.Context, id envelopes.ID, destination envelopes.IDer) (err error) {
	contents, err := dl.Fetch(ctx, id)
	if err != nil {
		return
	}

	return json.Unmarshal(contents, destination)
}
