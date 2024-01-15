package filesystem

import (
	"os"
	"strings"
	"testing"

	"github.com/marstr/envelopes"
)

func Test_segmentNormalizedName(t *testing.T) {
	testCases := []normalizedBankRecordID{
		normalizeBankRecordID("20201212 575073 2,000 202,012,128,756"),
	}

	for _, tc := range testCases {
		segments := segmentNormalizedName(tc)
		t.Logf("%q got split into %d segments", string(tc), len(segments))
		if cap(segments) != len(segments) {
			t.Errorf("too large of a capacity encountered")
		}

		joined := strings.Join(segments, "")
		if joined != string(tc) {
			t.Errorf("round trip failed.\n\twant: %s\n\tgot:  %s", string(tc), joined)
		}
	}
}

func TestFilesystemBankRecordIDIndex_AppendBankRecordID(t *testing.T) {
	bankRecordId := envelopes.BankRecordID("20201212 575073 2,000 202,012,128,756")
	tempDir, err := os.MkdirTemp("", "bankRecordID_index_test")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(tempDir)
	t.Logf("tempDir: %q", tempDir)

	subject := FilesystemBankRecordIDIndex{
		Root: tempDir,
	}

	var firstTransaction envelopes.ID
	firstTransaction[0] = 1

	var preExists bool
	preExists, err = subject.HasBankRecordId(bankRecordId)
	if err != nil {
		t.Error(err)
		return
	}
	if preExists {
		t.Errorf("bank record ID %q claims to already exist", bankRecordId)
		return
	}

	err = subject.AppendBankRecordID(bankRecordId, firstTransaction)
	if err != nil {
		t.Error(err)
		return
	}

	var postExists bool
	postExists, err = subject.HasBankRecordId(bankRecordId)

	if err != nil {
		t.Error(err)
		return
	}

	if !postExists {
		t.Error("bank id should have been found")
	}

	var secondTransaction envelopes.ID
	secondTransaction[0] = 2

	err = subject.AppendBankRecordID(bankRecordId, secondTransaction)
	if err != nil {
		t.Error(err)
		return
	}

	var associatedTransactions <-chan envelopes.ID
	associatedTransactions, err = subject.listTransactions(bankRecordId)
	if err != nil {
		t.Error(err)
		return
	}

	expected := map[envelopes.ID]struct{}{
		firstTransaction:  {},
		secondTransaction: {},
	}
	for seen := range associatedTransactions {
		_, ok := expected[seen]
		if ok {
			delete(expected, seen)
		} else {
			t.Errorf("unexpected transaction encountered: %s", seen)
		}
	}

	if remaining := len(expected); remaining > 0 {
		t.Errorf("didn't encounter %d expected transactions", remaining)
	}
}

func TestFilesystemBankRecordIDIndex_WriteBankRecordID(t *testing.T) {
	bankRecordId := envelopes.BankRecordID("20201212 575073 2,000 202,012,128,756")
	tempDir, err := os.MkdirTemp("", "bankRecordID_index_test")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(tempDir)
	t.Logf("tempDir: %q", tempDir)

	subject := FilesystemBankRecordIDIndex{
		Root: tempDir,
	}

	var firstTransaction envelopes.ID
	firstTransaction[0] = 1

	var preExists bool
	preExists, err = subject.HasBankRecordId(bankRecordId)
	if err != nil {
		t.Error(err)
		return
	}
	if preExists {
		t.Errorf("bank record ID %q claims to already exist", bankRecordId)
		return
	}

	err = subject.WriteBankRecordID(bankRecordId, firstTransaction)
	if err != nil {
		t.Error(err)
		return
	}

	var postExists bool
	postExists, err = subject.HasBankRecordId(bankRecordId)

	if err != nil {
		t.Error(err)
		return
	}

	if !postExists {
		t.Error("bank id should have been found")
	}

	var secondTransaction envelopes.ID
	secondTransaction[0] = 2

	err = subject.WriteBankRecordID(bankRecordId, secondTransaction)
	if err != nil {
		t.Error(err)
		return
	}

	var associatedTransactions <-chan envelopes.ID
	associatedTransactions, err = subject.listTransactions(bankRecordId)
	if err != nil {
		t.Error(err)
		return
	}

	expected := map[envelopes.ID]struct{}{
		secondTransaction: {},
	}
	for seen := range associatedTransactions {
		_, ok := expected[seen]
		if ok {
			delete(expected, seen)
		} else {
			t.Errorf("unexpected transaction encountered: %s", seen)
		}
	}

	if remaining := len(expected); remaining > 0 {
		t.Errorf("didn't encounter %d expected transactions", remaining)
	}
}
