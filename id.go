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

package envelopes

import (
	"bytes"
	"encoding/hex"
	"sync"
)

var identityBuilders = &sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// ID is a 20 byte array containing a SHA1 hash of an object.
type ID [20]byte

// IDer exposes a mechanism for
type IDer interface {
	ID() ID
}

// Equal determines whether or not two IDs are equivalent.
func (id ID) Equal(other ID) bool {
	for i, b := range id {
		if b != other[i] {
			return false
		}
	}
	return true
}

func (id ID) String() string {
	marshaled, _ := id.MarshalText()
	return string(marshaled)
}

// MarshalText produces a 20-byte hexadecimal text representation
// of this ID.
func (id ID) MarshalText() (results []byte, err error) {
	var raw [2 * len(id)]byte
	results = raw[:]
	hex.Encode(results, id[:])
	return
}

// UnmarshalText takes a text representation of an ID and reads it
// into a more usable format.
func (id *ID) UnmarshalText(content []byte) (err error) {
	// Truncate content to size to trim trailing whitespace if present.
	_, err = hex.Decode(id[:], content[:2*len(id)])
	return
}
