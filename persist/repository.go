/*
 * Copyright {YEAR} Martin Strobel
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package persist

// BareRepositoryReader indicates that a struct is able to read objects like envelopes.Budget, envelopes.Transaction,
// and branch instances from a repository. However, it does not indicate that the repository has a current working copy.
type BareRepositoryReader interface {
	Loader
	BranchReader
	BranchLister
}

// BareRepositoryWriter indicates that a struct is able to add objects like envelopes.Budget, envelopes.Transaction,
// and branch instances to a repository. However, it does not indicate that the repository has a current working copy.
type BareRepositoryWriter interface {
	Writer
	BranchWriter
}

// BareRepositoryReaderWriter indicates that a struct has the ability to both read and write
type BareRepositoryReaderWriter interface {
	BareRepositoryReader
	BareRepositoryWriter
}

// RepositoryReader indicates that a structs has all the functionality of a BareRepositoryReader, plus a CurrentReader.
type RepositoryReader interface {
	BareRepositoryReader
	CurrentReader
}

// RepositoryWriter has all the functionality of a BareRepositoryWriter, plus the ability to update Current.
type RepositoryWriter interface {
	BareRepositoryWriter
	CurrentWriter
}

// RepositoryReaderWriter has all the functionality of a RepositoryReader and RepositoryWriter.
type RepositoryReaderWriter interface {
	RepositoryReader
	RepositoryWriter
}
