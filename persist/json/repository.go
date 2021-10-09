package json

import (
	"github.com/marstr/envelopes/persist"
	"github.com/marstr/envelopes/persist/filesystem"
	"os"
)

type FileSystemRepository struct {
	filesystem.FileSystem
	LoaderV2
	WriterV2
}

type FileSystemRepositoryOption func(repository *FileSystemRepository) error

// FileSystemRepositoryUseCache creates a FileSystemRepositoryOption that will first look for a budget object an in
// memory cache before proceeding to the filesystem. The cache is limited in size to the provided number of budget
// objects.
func FileSystemRepositoryUseCache(capacity uint) FileSystemRepositoryOption {
	cache := persist.NewCache(capacity)
	return func(repository *FileSystemRepository) error {
		repository.LoaderV2.Loopback = cache
		repository.WriterV2.Loopback = cache
		return nil
	}
}

// FileSystemRepositoryFileMode creates a FileSystemRepositoryOption that changes the permissions that will be used for
// newly created files as they are written.
func FileSystemRepositoryFileMode(mode os.FileMode) FileSystemRepositoryOption {
	return func(repository *FileSystemRepository) error {
		repository.FileSystem.CreatePermissions = mode
		return nil
	}
}

// NewFileSystemRepository constructs an persist.BareRepositoryReaderWriter that specifically stores JSON objects on a
// local filesystem.
func NewFileSystemRepository(root string, options ...FileSystemRepositoryOption) (*FileSystemRepository, error) {
	fs := filesystem.FileSystem{
		Root: root,
	}

	retval := &FileSystemRepository{
		FileSystem: fs,
		LoaderV2:     LoaderV2{
			Fetcher:  fs,
			Loopback: nil,
		},
		WriterV2:     WriterV2{
			Stasher:  fs,
			Loopback: nil,
		},
	}

	for i := range options {
		if err := options[i](retval); err != nil {
			return nil, err
		}
	}

	return retval, nil
}