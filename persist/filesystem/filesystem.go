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

// Package persist defines the basic requirements that the object model expects
// in order to save and load state. The object model expects to be spun up
// and down frequently.
package filesystem

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/marstr/envelopes"
	"github.com/marstr/envelopes/persist"

	"github.com/marstr/collection/v2"
	"github.com/mitchellh/go-homedir"
)

// ObjectsDir is the name of the directory that holds marshaled IDer objects.
const ObjectsDir = "objects"

// FileSystem allows an easy mechanism for reading and writing raw Budget related objects to and from a hard drive.
type FileSystem struct {
	Root              string
	CreatePermissions os.FileMode
	ObjectLayout      uint

	// Logger receives persistence diagnostics. A nil logger disables logging.
	Logger *slog.Logger
}

var _ envelopes.DiagnosticLoggerProvider = FileSystem{}

// DiagnosticLogger supplies the configured logger to higher-level persistence operations.
func (fs FileSystem) DiagnosticLogger() *slog.Logger {
	return fs.Logger
}

func (fs FileSystem) debug(ctx context.Context, operation, stage, loc string, err error, attrs ...slog.Attr) {
	if fs.Logger == nil || !fs.Logger.Enabled(ctx, slog.LevelDebug) {
		return
	}
	attrs = append(attrs, slog.String("operation", operation), slog.String("stage", stage))
	if loc != "" {
		attrs = append(attrs, slog.String("path", loc))
	}
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
	}
	fs.Logger.LogAttrs(ctx, slog.LevelDebug, "filesystem operation", attrs...)
}

func (fs FileSystem) close(ctx context.Context, operation, loc string, handle io.Closer) {
	if err := handle.Close(); err != nil {
		fs.warn(ctx, operation, "close", loc, err)
	}
}

func (fs FileSystem) warn(ctx context.Context, operation, stage, loc string, err error) {
	if fs.Logger != nil {
		fs.Logger.WarnContext(ctx, "filesystem failure not returned to caller",
			"operation", operation, "stage", stage, "path", loc, "error", err)
	}
}

func (fs FileSystem) getCreatePermissions() os.FileMode {
	if fs.CreatePermissions == 0 {
		return 0660
	}
	return fs.CreatePermissions
}

// Current fetches the RefSpec that was most recently used to populate the index.
func (fs FileSystem) Current(ctx context.Context) (persist.RefSpec, error) {
	p, err := fs.currentPath()
	if err != nil {
		fs.debug(ctx, "current", "path", p, err)
		return "", err
	}

	raw, err := os.ReadFile(p)
	fs.debug(ctx, "current", "read", p, err)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(string(raw))
	return persist.RefSpec(trimmed), nil
}

// SetCurrent replaces the current pointer to the most recent Transaction with a given RefSpec. For instance, this
// should be used to change which branch is currently checked-out.
func (fs FileSystem) SetCurrent(ctx context.Context, current persist.RefSpec) error {
	p, err := fs.currentPath()
	if err != nil {
		fs.debug(ctx, "set_current", "path", p, err)
		return err
	}
	err = os.WriteFile(p, []byte(current), fs.getCreatePermissions())
	fs.debug(ctx, "set_current", "write", p, err)
	return err
}

// Fetch is able to read into memory the marshaled form of a Budget related object.
//
// See Also:
// - FileSystem.Stash
func (fs FileSystem) Fetch(ctx context.Context, id envelopes.ID) ([]byte, error) {
	p, err := fs.path(id)
	if err != nil {
		fs.debug(ctx, "fetch", "path", p, err, slog.Any("object_id", id))
		return nil, err
	}
	raw, err := os.ReadFile(p)
	fs.debug(ctx, "fetch", "read", p, err, slog.Any("object_id", id), slog.Int("bytes", len(raw)))
	return raw, err
}

// FetchReadCloser returns a ReadCloser that can be used to read the contents of a file
// holding a marshaled IDer.
//
// See Also:
// - FileSystem.StashReadCloser
func (fs FileSystem) FetchReadCloser(ctx context.Context, id envelopes.ID) (io.ReadCloser, error) {
	p, err := fs.path(id)
	if err != nil {
		fs.debug(ctx, "fetch_stream", "path", p, err, slog.Any("object_id", id))
		return nil, err
	}
	handle, err := os.Open(p)
	fs.debug(ctx, "fetch_stream", "open", p, err, slog.Any("object_id", id))
	return handle, err
}

// Stash commits the provided payload to disk at a place that it can retreive again if asked for the ID specified here.
//
// See Also:
// - FileSystem.Fetch
func (fs FileSystem) Stash(ctx context.Context, id envelopes.ID, payload []byte) error {
	loc, err := fs.path(id)
	if err != nil {
		fs.debug(ctx, "stash", "path", loc, err, slog.Any("object_id", id))
		return err
	}

	err = os.MkdirAll(filepath.Dir(loc), fs.getCreatePermissions()|0110|os.ModeDir)
	if err != nil {
		fs.debug(ctx, "stash", "mkdir", loc, err, slog.Any("object_id", id))
		return err
	}

	err = os.WriteFile(loc, payload, fs.getCreatePermissions())
	fs.debug(ctx, "stash", "write", loc, err, slog.Any("object_id", id), slog.Int("bytes", len(payload)))
	return err
}

// StashReadCloser commits the provided payload to disk by streaming from the given ReadCloser.
// Unlike Stash, which accepts the full payload as a byte slice, this method does not buffer the
// entire contents in memory.
//
// See Also:
// - FileSystem.FetchReadCloser
func (fs FileSystem) StashReadCloser(ctx context.Context, id envelopes.ID, payload io.ReadCloser) error {
	loc, err := fs.path(id)
	if err != nil {
		fs.debug(ctx, "stash_stream", "path", loc, err, slog.Any("object_id", id))
		return err
	}

	err = os.MkdirAll(filepath.Dir(loc), fs.getCreatePermissions()|0110|os.ModeDir)
	if err != nil {
		fs.debug(ctx, "stash_stream", "mkdir", loc, err, slog.Any("object_id", id))
		return err
	}

	var handle io.WriteCloser
	handle, err = os.Create(loc)
	if err != nil {
		fs.debug(ctx, "stash_stream", "open", loc, err, slog.Any("object_id", id))
		return err
	}
	defer fs.close(ctx, "stash_stream", loc, handle)

	n, err := io.Copy(handle, payload)
	fs.close(ctx, "stash_stream_source", loc, payload)
	fs.debug(ctx, "stash_stream", "copy", loc, err, slog.Any("object_id", id), slog.Int64("bytes", n))

	return err
}

// currentPath fetches the name of the file containing the ID to the most up-to-date Transaction.
func (fs FileSystem) currentPath() (string, error) {
	exp, err := homedir.Expand(fs.Root)
	if err != nil {
		return "", err
	}
	return filepath.Join(exp, "current.txt"), nil
}

func (fs FileSystem) path(id envelopes.ID) (string, error) {
	exp, err := homedir.Expand(fs.Root)
	if err != nil {
		return "", err
	}

	switch fs.ObjectLayout {
	case 0:
		return filepath.Join(exp, ObjectsDir, id.String()+".json"), nil
	case 1:
		full := id.String()
		return filepath.Join(exp, ObjectsDir, full[:2], full[2:]+".json"), nil
	default:
		return "", fmt.Errorf("unrecognized object layout %v", fs.ObjectLayout)
	}
}

func (fs FileSystem) branchPath(name string) string {
	return filepath.Join(fs.Root, "refs", "heads", name)
}

// ReadBranch fetches the ID that a branch is pointing at.
func (fs FileSystem) ReadBranch(ctx context.Context, name string) (envelopes.ID, error) {
	branchLoc := fs.branchPath(name)
	var retval envelopes.ID
	handle, err := os.Open(branchLoc)
	if err != nil {
		fs.debug(ctx, "read_branch", "open", branchLoc, err)
		return retval, err
	}
	defer fs.close(ctx, "read_branch", branchLoc, handle)

	var contents [2 * cap(retval)]byte
	var n int
	n, err = handle.Read(contents[:])
	if err != nil {
		fs.debug(ctx, "read_branch", "read", branchLoc, err)
		return retval, err
	}

	if expected := cap(contents); n != expected {
		err = fmt.Errorf(
			"%s was not long enough to be a candidate for pointing to a Transaction ID (want: %v got: %v)",
			branchLoc,
			expected,
			n)
		fs.debug(ctx, "read_branch", "read", branchLoc, err)
		return retval, err
	}

	err = retval.UnmarshalText(contents[:])
	fs.debug(ctx, "read_branch", "parse", branchLoc, err)
	return retval, err
}

// WriteBranch sets a branch to be pointing at a particular ID.
func (fs FileSystem) WriteBranch(ctx context.Context, name string, id envelopes.ID) error {
	branchLoc := fs.branchPath(name)

	err := os.MkdirAll(filepath.Dir(branchLoc), fs.getCreatePermissions()|0110|os.ModeDir)
	if err != nil {
		fs.debug(ctx, "write_branch", "mkdir", branchLoc, err, slog.Any("object_id", id))
		return err
	}

	err = os.WriteFile(branchLoc, []byte(id.String()), fs.getCreatePermissions())
	fs.debug(ctx, "write_branch", "write", branchLoc, err, slog.Any("object_id", id))
	return err
}

// ListBranches fetches the distinct names of the branches that exist in a repository.
func (fs FileSystem) ListBranches(ctx context.Context) (<-chan string, error) {
	absRoot, err := filepath.Abs(filepath.Dir(fs.branchPath("any_branch_name")))
	if err != nil {
		fs.debug(ctx, "list_branches", "path", "", err)
		return nil, err
	}
	fs.debug(ctx, "list_branches", "start", absRoot, nil)

	dir := collection.Directory{
		Location: absRoot,
		Options:  collection.DirectoryOptionsExcludeDirectories | collection.DirectoryOptionsRecursive,
	}

	rawResults := dir.Enumerate(ctx)

	prefix := absRoot + "/"
	prefix = strings.Replace(prefix, "\\", "/", -1)
	castResults := make(chan string)
	go func() {
		defer close(castResults)
		count := 0
		defer func() {
			fs.debug(ctx, "list_branches", "enumerated", absRoot, ctx.Err(), slog.Int("count", count))
		}()

		for entry := range rawResults {
			entry = strings.Replace(entry, "\\", "/", -1)
			trimmed := strings.TrimPrefix(entry, prefix)
			select {
			case <-ctx.Done():
				return
			case castResults <- trimmed:
				count++
			}
		}
	}()

	return castResults, nil
}

// refsDir returns the absolute path to a refs subdirectory (e.g., "heads" or "tags").
func (fs FileSystem) refsDir(subdir string) (string, error) {
	return filepath.Abs(filepath.Join(fs.Root, "refs", subdir))
}

func (fs FileSystem) tagPath(name string) string {
	return filepath.Join(fs.Root, "refs", "tags", name)
}

// ReadTag fetches the Tag information including ID and comment.
func (fs FileSystem) ReadTag(ctx context.Context, name string) (persist.Tag, error) {
	tagLoc := fs.tagPath(name)
	var retval persist.Tag
	handle, err := os.Open(tagLoc)
	if err != nil {
		fs.debug(ctx, "read_tag", "open", tagLoc, err)
		return retval, err
	}
	defer fs.close(ctx, "read_tag", tagLoc, handle)

	// Read the ID from the first fixed number of bytes (same as branches)
	var idBytes [2 * cap(retval.ID)]byte
	var n int
	n, err = handle.Read(idBytes[:])
	if err != nil {
		fs.debug(ctx, "read_tag", "read", tagLoc, err)
		return retval, err
	}

	if expected := cap(idBytes); n != expected {
		err = fmt.Errorf(
			"%s was not long enough to be a candidate for pointing to a Transaction ID (want: %v got: %v)",
			tagLoc,
			expected,
			n)
		fs.debug(ctx, "read_tag", "read", tagLoc, err)
		return retval, err
	}

	// Parse the ID
	err = retval.ID.UnmarshalText(idBytes[:])
	if err != nil {
		fs.debug(ctx, "read_tag", "parse", tagLoc, err)
		return retval, err
	}

	// Read the rest of the file as the comment
	// First, consume the newline character(s) after the ID
	var newlineBuf [2]byte
	n, err = handle.Read(newlineBuf[:])
	if err != nil && err != io.EOF {
		fs.warn(ctx, "read_tag", "read_comment", tagLoc, err)
		fs.debug(ctx, "read_tag", "read_comment", tagLoc, nil)
		return persist.Tag{ID: retval.ID}, nil // No comment, just return the ID
	}

	// Handle both Unix (\n) and Windows (\r\n) line endings
	startOffset := 0
	if n > 0 && newlineBuf[0] == '\r' && n > 1 && newlineBuf[1] == '\n' {
		// Windows line ending, both bytes consumed
		startOffset = 2
	} else if n > 0 && newlineBuf[0] == '\n' {
		// Unix line ending, one byte consumed
		startOffset = 1
	} else {
		// No newline found, no comment
		fs.debug(ctx, "read_tag", "read_comment", tagLoc, nil)
		return persist.Tag{ID: retval.ID}, nil
	}

	// Read any remaining bytes as the comment
	if n > startOffset {
		// We read more than just the newline, prepend those bytes to the comment
		remainingBytes := newlineBuf[startOffset:n]
		commentBytes, readErr := io.ReadAll(handle)
		if readErr != nil && readErr != io.EOF {
			fs.debug(ctx, "read_tag", "read_comment", tagLoc, readErr)
			return retval, readErr
		}
		retval.Comment = string(remainingBytes) + string(commentBytes)
	} else {
		// Read the rest of the file
		commentBytes, readErr := io.ReadAll(handle)
		if readErr != nil && readErr != io.EOF {
			fs.debug(ctx, "read_tag", "read_comment", tagLoc, readErr)
			return retval, readErr
		}
		retval.Comment = string(commentBytes)
	}

	fs.debug(ctx, "read_tag", "read_comment", tagLoc, nil)
	return retval, nil
}

// WriteTag sets a tag with the given ID and comment.
func (fs FileSystem) WriteTag(ctx context.Context, name string, tag persist.Tag) error {
	tagLoc := fs.tagPath(name)

	err := os.MkdirAll(filepath.Dir(tagLoc), fs.getCreatePermissions()|0110|os.ModeDir)
	if err != nil {
		fs.debug(ctx, "write_tag", "mkdir", tagLoc, err, slog.Any("object_id", tag.ID))
		return err
	}

	// Create the file
	handle, err := os.Create(tagLoc)
	if err != nil {
		fs.debug(ctx, "write_tag", "open", tagLoc, err, slog.Any("object_id", tag.ID))
		return err
	}
	defer fs.close(ctx, "write_tag", tagLoc, handle)

	// Write the ID
	_, err = handle.WriteString(tag.ID.String())
	if err != nil {
		fs.debug(ctx, "write_tag", "write", tagLoc, err, slog.Any("object_id", tag.ID))
		return err
	}

	// Write the comment if present
	if tag.Comment != "" {
		// Write newline separator
		_, err = handle.WriteString("\n")
		if err != nil {
			fs.debug(ctx, "write_tag", "write", tagLoc, err, slog.Any("object_id", tag.ID))
			return err
		}

		// Write the comment
		_, err = handle.WriteString(tag.Comment)
		if err != nil {
			fs.debug(ctx, "write_tag", "write", tagLoc, err, slog.Any("object_id", tag.ID))
			return err
		}
	}

	fs.debug(ctx, "write_tag", "write", tagLoc, nil, slog.Any("object_id", tag.ID))
	return nil
}

// ListTags fetches the distinct names of the tags that exist in a repository.
func (fs FileSystem) ListTags(ctx context.Context) (<-chan string, error) {
	absRoot, err := fs.refsDir("tags")
	if err != nil {
		fs.debug(ctx, "list_tags", "path", "", err)
		return nil, err
	}
	fs.debug(ctx, "list_tags", "start", absRoot, nil)

	dir := collection.Directory{
		Location: absRoot,
		Options:  collection.DirectoryOptionsExcludeDirectories | collection.DirectoryOptionsRecursive,
	}

	rawResults := dir.Enumerate(ctx)

	prefix := absRoot + "/"
	prefix = strings.Replace(prefix, "\\", "/", -1)
	castResults := make(chan string)
	go func() {
		defer close(castResults)
		count := 0
		defer func() {
			fs.debug(ctx, "list_tags", "enumerated", absRoot, ctx.Err(), slog.Int("count", count))
		}()

		for entry := range rawResults {
			entry = strings.Replace(entry, "\\", "/", -1)
			trimmed := strings.TrimPrefix(entry, prefix)
			select {
			case <-ctx.Done():
				return
			case castResults <- trimmed:
				count++
			}
		}
	}()

	return castResults, nil
}
