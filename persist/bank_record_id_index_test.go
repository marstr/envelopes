package persist

import (
	"io/ioutil"
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
	bankRecordId := BankRecordID("20201212 575073 2,000 202,012,128,756")
	tempDir, err := ioutil.TempDir("", "bankRecordID_index_test")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(tempDir)
	t.Logf("tempDir: %q", tempDir)

	subject := FilesystemBankRecordIDIndex{
		Root: tempDir,
	}

	var transactionId envelopes.ID
	transactionId[0] = 1

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

	err = subject.AppendBankRecordID(bankRecordId, transactionId)
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
}
