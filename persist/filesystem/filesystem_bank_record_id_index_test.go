package filesystem

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marstr/envelopes"
)

func TestFilesystemBankRecordIDIndex_Logging(t *testing.T) {
	var logs, defaultLogs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&defaultLogs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	index := FilesystemBankRecordIDIndex{Root: t.TempDir()}
	const bankID envelopes.BankRecordID = "private-bank-record-sentinel"
	id := envelopes.ID{1}
	if err := index.AppendBankRecordID(bankID, id); err != nil {
		t.Fatal(err)
	}
	if defaultLogs.Len() != 0 {
		t.Fatal("nil index logger used the default logger")
	}
	index.Logger = slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	found, err := index.HasBankRecordId(bankID)
	if err != nil || !found {
		t.Fatalf("index lookup: %v, %v", found, err)
	}
	if err := index.AppendBankRecordID(bankID, envelopes.ID{2}); err != nil {
		t.Fatal(err)
	}
	if err := index.WriteBankRecordID(bankID, id); err != nil {
		t.Fatal(err)
	}
	transactions, err := index.listTransactions(bankID)
	if err != nil {
		t.Fatal(err)
	}
	for range transactions {
	}
	if err := index.ClearBankRecordID(bankID); err != nil {
		t.Fatal(err)
	}
	if err := index.ClearBankRecordID(bankID); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"has_bank_record", "append_bank_record", "write_bank_record", "clear_bank_record"} {
		if !strings.Contains(logs.String(), `"operation":"`+operation+`"`) {
			t.Fatalf("missing %s diagnostic: %s", operation, logs.String())
		}
	}
	if strings.Contains(logs.String(), `"level":"WARN"`) {
		t.Fatalf("normal EOF or missing index entry produced a warning: %s", logs.String())
	}
	assertIndexLogsPrivate(t, &logs, bankID)

	logs.Reset()
	blockedRoot := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blockedRoot, nil, 0600); err != nil {
		t.Fatal(err)
	}
	index.Root = blockedRoot
	if err := index.WriteBankRecordID(bankID, id); err == nil {
		t.Fatal("expected index mkdir failure")
	}
	if !strings.Contains(logs.String(), `"stage":"mkdir"`) || !strings.Contains(logs.String(), `"error":`) {
		t.Fatalf("missing failure context: %s", logs.String())
	}
	assertIndexLogsPrivate(t, &logs, bankID)

	logs.Reset()
	index.Root = t.TempDir()
	filename, err := index.bankRecordIdFilename(bankID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		t.Fatal(err)
	}
	const invalidLine = "private-invalid-transaction-line-sentinel-extra"
	if err := os.WriteFile(filename, []byte(invalidLine+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	transactions, err = index.listTransactions(bankID)
	if err != nil {
		t.Fatal(err)
	}
	for range transactions {
	}
	if !strings.Contains(logs.String(), `"level":"WARN"`) || !strings.Contains(logs.String(), `"stage":"parse"`) {
		t.Fatalf("missing asynchronous parse diagnostic: %s", logs.String())
	}
	if strings.Contains(logs.String(), invalidLine) {
		t.Fatal("logged invalid index contents")
	}
	assertIndexLogsPrivate(t, &logs, bankID)
}

func assertIndexLogsPrivate(t *testing.T, logs *bytes.Buffer, id envelopes.BankRecordID) {
	t.Helper()
	for _, private := range append(segmentNormalizedName(normalizeBankRecordID(id)), string(id)) {
		if strings.Contains(logs.String(), private) {
			t.Fatal("logged bank-record ID or encoded filename")
		}
	}
	if strings.Contains(logs.String(), `"path":`) {
		t.Fatal("logged index path")
	}
}

func TestFilesystemBankRecordIDIndex_AsyncReadLogging(t *testing.T) {
	var logs bytes.Buffer
	index := FilesystemBankRecordIDIndex{
		Root:   t.TempDir(),
		Logger: slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	const bankID envelopes.BankRecordID = "private-bank-read-error-sentinel"
	filename, err := index.bankRecordIdFilename(bankID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filename, 0700); err != nil {
		t.Fatal(err)
	}
	transactions, err := index.listTransactions(bankID)
	if err != nil {
		t.Fatal(err)
	}
	for range transactions {
	}
	if !strings.Contains(logs.String(), `"level":"WARN"`) || !strings.Contains(logs.String(), `"stage":"read"`) {
		t.Fatalf("missing asynchronous read diagnostic: %s", logs.String())
	}
	assertIndexLogsPrivate(t, &logs, bankID)
}

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
