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

package envelopes_test

import (
	"context"
	"crypto/rand"
	"io"
	"os"
	"testing"
	"time"

	"github.com/marstr/envelopes"
)

func openFishingRules(_ context.Context) (io.ReadCloser, error) {
	return os.Open("./2025ORFW.pdf")
}

func TestAttachment_Equal_dontTestContents(t *testing.T) {
	var ctx context.Context
	deadline, ok := t.Deadline()
	if ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	left := envelopes.Attachment{
		Extension: "pdf",
		Contents:  openFishingRules,
		Comment:   "Same same but different",
	}

	var err error
	left.ContentID, err = left.ContentSHA1(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	right := left
	right.Contents = func(_ context.Context) (io.ReadCloser, error) {
		return io.NopCloser(rand.Reader), nil
	}

	if !left.Equal(right) {
		t.Error("these attachements are identical except for the actual contents field, but that shouldn't be tested as part of the Equal function")
	}
}

func TestAttachment_ContentSHA1(t *testing.T) {
	ctx := context.Background()
	deadline, ok := t.Deadline()
	if ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(context.Background(), deadline)
		defer cancel()
	}

	subject := envelopes.Attachment{
		Extension: "pdf",
		Contents:  openFishingRules,
	}

	expected := "2d046b90a87e82ff35d972d5855f57c548f23129"

	result, err := subject.ContentSHA1(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	got := result.String()
	if got != expected {
		t.Errorf("incorrect SHA1 hash:\n\twant %q\n\t got %q", expected, got)
	}
}

func TestAttachment_ContentSHA1_respectContext(t *testing.T) {
	var ctx context.Context
	var cancel context.CancelFunc
	deadline, ok := t.Deadline()
	if ok {
		ctx, cancel = context.WithDeadline(context.Background(), deadline.Add(-5*time.Second))
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	subject := envelopes.Attachment{
		Contents: func(ctx context.Context) (io.ReadCloser, error) {
			return io.NopCloser(rand.Reader), nil
		},
	}

	testDelay := 50 * time.Millisecond
	gracePeriod := 10 * testDelay

	// Pvt. Joker is a character from Full Metal Jacket (1987) who ironically wears the slogan "born to kill" on his helmet.
	joker, cancelJoker := context.WithTimeout(ctx, testDelay)
	defer cancelJoker()

	// Pvt. Pyle is the other main character from Full Metal Jacket (1987), who can't hack it after the grace period.
	pyle, cancelPyle := context.WithTimeout(context.Background(), gracePeriod)
	defer cancelPyle()

	failureReason := make(chan error)

	go func() {
		hash, err := subject.ContentSHA1(joker)

		if !hash.Equal(envelopes.ID{}) || err == nil {
			t.Errorf("this test somehow read to the end of an inexhaustible reader?")
		}
		failureReason <- err
	}()

	select {
	case <-pyle.Done():
		t.Errorf("function did not fail and yield control within the time allotted")
	case got := <-failureReason:
		t.Log(got)
	}
}
