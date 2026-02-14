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

func TestAttachment_ContentSHA1(t *testing.T) {
	var ctx context.Context
	var cancel context.CancelFunc
	deadline, ok := t.Deadline()
	if ok {
		ctx, cancel = context.WithDeadline(context.Background(), deadline)
	} else {
		ctx = context.Background()
	}
	defer cancel()

	subject := envelopes.Attachment{
		Extension: "pdf",
		Contents: func(_ context.Context) (io.ReadCloser, error) {
			return os.Open("./2025ORFW.pdf")
		},
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
		ctx = context.Background()
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
			t.Errorf("this test somehow read to the end of an inexhaustable reader?")
		}
		failureReason <- err
	}()

	select {
	case <-pyle.Done():
		t.Errorf("function did not fail and yield control within the time alloted")
	case got := <-failureReason:
		t.Log(got)
	}
}
