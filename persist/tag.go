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

// TagReader indicates that a type is capable of discovering the envelopes.Transaction that a tag points at.
type TagReader interface {
	ReadTag(ctx context.Context, name string) (envelopes.ID, error)
}

// TagWriter indicates that a type is capable of setting the envelopes.Transaction that a tag points at.
type TagWriter interface {
	WriteTag(ctx context.Context, name string, id envelopes.ID) error
}

// TagReaderWriter indicates that a type has both the capabilities of a TagReader and TagWriter.
type TagReaderWriter interface {
	TagReader
	TagWriter
}

// TagLister are able to find all tags in a repository.
type TagLister interface {
	ListTags(ctx context.Context) (<-chan string, error)
}
