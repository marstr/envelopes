// Copyright 2020 Martin Strobel
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

package filesystem

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/marstr/envelopes/persist"
	"os"
	"path"
	"strings"

	"github.com/marstr/envelopes"
)

type ErrEmptyBankRecordID struct{}

type normalizedBankRecordID string

func (err ErrEmptyBankRecordID) Error() string {
	return "the default, empty bank record ID is not a valid argument"
}

type FilesystemBankRecordIDIndex struct {
	Root            string
	DecoratedWriter persist.Writer
}

// Converts an arbitrary bank transaction ID to something that's deterministic and only contains characters safe for
// using in filenames i.e. an RFC 4648 Base 64 URL encoded string: https://tools.ietf.org/html/rfc4648#section-5
func normalizeBankRecordID(id envelopes.BankRecordID) normalizedBankRecordID {
	return normalizedBankRecordID(base64.URLEncoding.EncodeToString([]byte(id)))
}

func segmentNormalizedName(id normalizedBankRecordID) []string {
	const segmentLength = 8
	remaining := string(id)
	retval := make([]string, 0, (len(id)/segmentLength)+1)
	for len(remaining) > segmentLength {
		retval = append(retval, remaining[:segmentLength])
		remaining = remaining[segmentLength:]
	}
	retval = append(retval, remaining)
	return retval
}

// Write associates the given Transaction with it's BankRecordID if applicable, then passes the call along to the next
// Writer.
//
// If subject is not a Transaction, or subject is a Transaction but does not have a BankRecordID, the association step
// is skipped altogether, and this continues to call the DecoratedWriter.
//
// If DecoratedWriter is nil, the association step will still happen if applicable, but then nothing more happens.
func (index FilesystemBankRecordIDIndex) Write(ctx context.Context, subject envelopes.IDer) error {
	transaction, ok := subject.(envelopes.Transaction)
	if ok && transaction.RecordID != "" {
		err := index.AppendBankRecordID(transaction.RecordID, transaction.ID())
		if err != nil {
			return err
		}
	}

	if index.DecoratedWriter != nil {
		return index.DecoratedWriter.Write(ctx, subject)
	}

	return nil
}

func (index FilesystemBankRecordIDIndex) bankRecordIdFilename(bankRecordID envelopes.BankRecordID) (string, error) {
	if bankRecordID == "" {
		return "", ErrEmptyBankRecordID{}
	}
	normalized := normalizeBankRecordID(bankRecordID)
	segmented := segmentNormalizedName(normalized)
	dirName := strings.Join(segmented, string(os.PathSeparator))
	dirName = path.Join(index.Root, dirName)
	return dirName + ".txt", nil
}

// HasBankRecordId returns true if this repository has at least one Transaction associated with a given BankRecordID.
func (index FilesystemBankRecordIDIndex) HasBankRecordId(id envelopes.BankRecordID) (bool, error) {
	var fileName string
	var err error
	fileName, err = index.bankRecordIdFilename(id)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(fileName)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// ClearBankRecordID disassociates all transactions from this BankRecordID.
func (index FilesystemBankRecordIDIndex) ClearBankRecordID(bankRecordID envelopes.BankRecordID) error {
	var fileName string
	var err error
	fileName, err = index.bankRecordIdFilename(bankRecordID)
	if err != nil {
		return err
	}
	err = os.Remove(fileName)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// WriteBankRecordID replaces the list of Transactions associated with a BankRecordID.
func (index FilesystemBankRecordIDIndex) WriteBankRecordID(bankRecordID envelopes.BankRecordID, transactionIDs ...envelopes.ID) error {
	return index.processBankRecordID(os.O_TRUNC|os.O_CREATE|os.O_WRONLY, bankRecordID, transactionIDs...)
}

// AppendBankRecordID adds to the list of Transactions associated with a BankRecordID.
func (index FilesystemBankRecordIDIndex) AppendBankRecordID(bankRecordID envelopes.BankRecordID, transactionIDs ...envelopes.ID) error {
	return index.processBankRecordID(os.O_APPEND|os.O_CREATE|os.O_WRONLY, bankRecordID, transactionIDs...)
}

func (index FilesystemBankRecordIDIndex) processBankRecordID(flag int, bankRecordID envelopes.BankRecordID, transactionIds ...envelopes.ID) error {
	if len(transactionIds) == 0 {
		return nil
	}

	var err error
	var fileName string
	fileName, err = index.bankRecordIdFilename(bankRecordID)
	if err != nil {
		return err
	}

	err = os.MkdirAll(path.Dir(fileName), os.ModePerm)
	if err != nil {
		return err
	}
	var handle *os.File
	handle, err = os.OpenFile(fileName, flag, os.ModePerm)
	if err != nil {
		return err
	}
	defer handle.Close()

	for _, transaction := range transactionIds {
		_, err = fmt.Fprintln(handle, transaction.String())
		if err != nil {
			return err
		}
	}
	return nil
}

// listTransactions enumerates all of the envelopes.Transaction IDs that are associated with a particular
// envelopes.BankRecordId.
//
// It's still private because I don't want to commit to an interface yet, i.e. this is an inappropriate use of channels.
func (index FilesystemBankRecordIDIndex) listTransactions(id envelopes.BankRecordID) (<-chan envelopes.ID, error) {
	var err error
	var fileName string
	fileName, err = index.bankRecordIdFilename(id)
	if err != nil {
		return nil, err
	}

	var handle *os.File
	handle, err = os.Open(fileName)
	if err != nil {
		return nil, err
	}

	retval := make(chan envelopes.ID)
	go func() {
		defer close(retval)
		defer handle.Close()

		reader := bufio.NewReader(handle)
		for {
			var line []byte
			var current envelopes.ID
			line, err = reader.ReadBytes('\n')
			if err != nil {
				// this is why channels are so bad for this
				return
			}

			err = current.UnmarshalText(line)
			if err != nil {
				// this is why channels are so bad for this
				return
			}
			retval <- current
		}
	}()

	return retval, nil
}
