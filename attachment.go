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

// Attachment stores a path to a file
type Attachment struct {
	Extension string
	ContentId ID
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

func (a Attachment) MarshalText() ([]byte, error) {
	var err error

	identityBuilder := identityBuilders.Get().(*bytes.Buffer)
	identityBuilder.Reset()
	defer identityBuilders.Put(identityBuilder)

	_, err = fmt.Fprintf(identityBuilder, "ext %s\n", a.Extension)
	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintf(identityBuilder, "contentHash %s\n", a.ContentId)
	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintf(identityBuilder, "comment %s\n", a.Extension)
	if err != nil {
		return nil, err
	}

	return identityBuilder.Bytes(), nil
}

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
