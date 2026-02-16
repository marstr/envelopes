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
	"context"
	"crypto/sha1"
	"fmt"
	"hash"
	"io"

	"github.com/marstr/units/data"
)

// Attachment represents a file-like payload, including metadata and a function for obtaining its contents.
type Attachment struct {
	Extension string
	ContentID ID
	Contents  func(ctx context.Context) (io.ReadCloser, error)
	Comment   string
}

func (a Attachment) ID() ID {
	raw, err := a.MarshalText()
	if err != nil {
		return ID{}
	}

	return sha1.Sum(raw)
}

// Equal compares all of the metadata fields in Attachment to ensure they are the
// same, but does not read the actual Contents to guarantee that they are the same.
// To achieve that, you can use this Equal function, then compare the results of
// calling ContentSHA1 on each Attachment.
func (a Attachment) Equal(other Attachment) bool {
	return a.Extension == other.Extension &&
		a.ContentID.Equal(other.ContentID) &&
		a.Comment == other.Comment
}

// MarshalText generates text that is enough to capture fields except
// for Contents, which should be marshaled and stored independently.
func (a Attachment) MarshalText() ([]byte, error) {
	var err error

	identityBuilder := identityBuilders.Get().(*bytes.Buffer)
	identityBuilder.Reset()
	defer identityBuilders.Put(identityBuilder)

	_, err = fmt.Fprintf(identityBuilder, "ext %s\n", a.Extension)
	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintf(identityBuilder, "contentHash %s\n", a.ContentID)
	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintf(identityBuilder, "comment %s\n", a.Comment)
	if err != nil {
		return nil, err
	}

	return identityBuilder.Bytes(), nil
}

// ContentSHA1 streams the content of the file in 5KB chunks and finds the SHA1 hash.
// The result of this function should match what is stored in Attachment.ContentID.
func (a Attachment) ContentSHA1(ctx context.Context) (ID, error) {
	var err error
	var reader io.ReadCloser
	var buffer [5 * data.Kilobyte]byte

	reader, err = a.Contents(ctx)
	if err != nil {
		return ID{}, err
	}
	defer reader.Close()

	hasher := hashers.Get().(hash.Hash)
	hasher.Reset()
	defer hashers.Put(hasher)

	for {
		var n int
		select {
		case <-ctx.Done():
			return ID{}, ctx.Err()
		default:
			// Intentionally Left Blank
		}

		n, err = reader.Read(buffer[:])

		if n > 0 {
			var p int
			p, err = hasher.Write(buffer[:n])
			if err != nil {
				return ID{}, err
			}
			if n != p {
				return ID{}, fmt.Errorf("buffer error while computing Attachment SHA1. Expected to write %d bytes actually wrote %d bytes", n, p)
			}
		}

		if err == io.EOF {
			var retval ID
			copy(retval[:], hasher.Sum(nil))
			return retval, nil
		}

		if err != nil {
			return ID{}, err
		}
	}
}
