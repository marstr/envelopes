package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/marstr/envelopes/persist"
	persistJson "github.com/marstr/envelopes/persist/json"

	"github.com/marstr/collection/v2"
)

// ConfigFilename is the path relative to the filesystem root where the JSON based configuration file can be found.
const ConfigFilename = "config.json"

type RepositoryConfigEntry struct {
	Format  string `json:"format"`
	Version uint   `json:"version"`
}

type RepositoryConfig struct {
	Objects         RepositoryConfigEntry `json:"objects"`
	ObjectLocations uint                  `json:"objectLocs"`
	Branches        RepositoryConfigEntry `json:"branches"`
}

const (
	FormatJson = "json"
)

// missingConfiguration should be used when opening a repository old enough that config files are not present.
var missingConfiguration = RepositoryConfig{
	Objects: RepositoryConfigEntry{
		Format:  FormatJson,
		Version: 1,
	},
}

// defaultConfiguration should be used when a new repository is being created, and no other version was specified.
var defaultConfiguration = RepositoryConfig{
	Objects: RepositoryConfigEntry{
		Format:  FormatJson,
		Version: 3,
	},
	ObjectLocations: 1,
}

type ErrUnsupportedConfiguration RepositoryConfig

func (err ErrUnsupportedConfiguration) Error() string {
	return "this version is not capable of loading the specified configuration"
}

type Repository struct {
	FileSystem
	persist.Loader
	persist.Writer
}

type RepositoryOption func(repository *Repository) error

// RepositoryLogger sets the logger used for persistence diagnostics. A nil logger disables logging.
func RepositoryLogger(logger *slog.Logger) RepositoryOption {
	return func(repository *Repository) error {
		repository.FileSystem.Logger = logger
		return nil
	}
}

// RepositoryFileMode creates a RepositoryOption that changes the permissions that will be used for newly created files
// as they are written.
func RepositoryFileMode(mode os.FileMode) RepositoryOption {
	return func(repository *Repository) error {
		repository.FileSystem.CreatePermissions = mode
		return nil
	}
}

// RepositoryObjectLoc creates a RepositoryOption that sets the layout of the object files in the filesystem.
func RepositoryObjectLoc(layout uint) RepositoryOption {
	return func(repository *Repository) error {
		if repository.FileSystem.ObjectLayout != defaultConfiguration.ObjectLocations {
			return fmt.Errorf("repository object layout is already set to %v", repository.FileSystem.ObjectLayout)
		}
		repository.FileSystem.ObjectLayout = layout
		return nil
	}
}

// OpenRepository creates a handle for interacting with an existing filesystem-based repository.
func OpenRepository(ctx context.Context, loc string, options ...RepositoryOption) (*Repository, error) {
	return openRepository(ctx, loc, nil, options...)
}

// OpenRepositoryWithCache creates a handle for interacting with an existing filesystem-based repository, but includes
// an in-memory cache that will reduce the number of disk reads needed. The parameter cacheSize is the number of budget
// objects that can fit in the cache. Cache misses are read from disk.
func OpenRepositoryWithCache(ctx context.Context, loc string, cacheSize uint, options ...RepositoryOption) (*Repository, error) {
	cache := persist.NewCache(cacheSize)
	return openRepository(ctx, loc, cache, options...)
}

func openRepository(ctx context.Context, loc string, cache *persist.Cache, options ...RepositoryOption) (*Repository, error) {
	var err error
	var config *RepositoryConfig
	var creatingRepo bool

	objDir := collection.Directory{
		Location: path.Join(loc, ObjectsDir),
	}

	if collection.Any[string](objDir) {
		config, err = LoadConfig(ctx, loc)
		if err != nil {
			return nil, err
		}
	} else {
		config = &defaultConfiguration
		creatingRepo = true
	}

	retval := Repository{
		FileSystem: FileSystem{
			Root:         loc,
			ObjectLayout: config.ObjectLocations,
		},
		Loader: nil,
		Writer: nil,
	}
	fs := &retval.FileSystem

	if config.Objects.Format == FormatJson {
		switch config.Objects.Version {
		case 1:
			if cache == nil {
				retval.Loader, err = persistJson.NewLoaderV1(fs)
				if err != nil {
					return nil, err
				}
				retval.Writer, err = persistJson.NewWriterV1(fs)
				if err != nil {
					return nil, err
				}
			} else {
				retval.Loader, err = persistJson.NewLoaderV1WithLoopback(fs, cache)
				if err != nil {
					return nil, err
				}
				retval.Writer, err = persistJson.NewWriterV1WithLoopback(fs, cache)
				if err != nil {
					return nil, err
				}
			}
		case 2:
			if cache == nil {
				retval.Loader, err = persistJson.NewLoaderV2(fs)
				if err != nil {
					return nil, err
				}
				retval.Writer, err = persistJson.NewWriterV2(fs)
				if err != nil {
					return nil, err
				}
			} else {
				retval.Loader, err = persistJson.NewLoaderV2WithLoopback(fs, cache)
				if err != nil {
					return nil, err
				}
				retval.Writer, err = persistJson.NewWriterV2WithLoopback(fs, cache)
				if err != nil {
					return nil, err
				}
			}
		case 3:
			if cache == nil {
				retval.Loader, err = persistJson.NewLoaderV3(fs)
				if err != nil {
					return nil, err
				}
				retval.Writer, err = persistJson.NewWriterV3(fs)
				if err != nil {
					return nil, err
				}
			} else {
				retval.Loader, err = persistJson.NewLoaderV3WithLoopback(fs, cache)
				if err != nil {
					return nil, err
				}
				retval.Writer, err = persistJson.NewWriterV3WithLoopback(fs, cache)
				if err != nil {
					return nil, err
				}
			}
		default:
			return nil, ErrUnsupportedConfiguration{}
		}
	} else {
		return nil, ErrUnsupportedConfiguration{}
	}

	if cache != nil {
		cache.Loader = retval.Loader
		cache.Writer = retval.Writer
		retval.Loader = cache
		retval.Writer = cache
	}

	for i := range options {
		err = options[i](&retval)
		if err != nil {
			return nil, err
		}
	}

	if !creatingRepo && fs.ObjectLayout != config.ObjectLocations {
		err = fmt.Errorf("cannot change an existing repository's object layout from %v to %v",
			config.ObjectLocations, fs.ObjectLayout)
		fs.debug(ctx, "open_repository", "options", fs.Root, err)
		return nil, err
	}

	fs.debug(ctx, "open_repository", "configuration", fs.Root, nil,
		slog.Bool("creating", creatingRepo),
		slog.String("format", config.Objects.Format),
		slog.Uint64("version", uint64(config.Objects.Version)),
		slog.Uint64("object_layout", uint64(fs.ObjectLayout)),
		slog.Bool("cache_enabled", cache != nil))

	if creatingRepo {
		effectiveConfig := *config
		effectiveConfig.ObjectLocations = fs.ObjectLayout
		err = writeConfig(ctx, fs.Root, &effectiveConfig, fs.getCreatePermissions())
		fs.debug(ctx, "write_config", "write", path.Join(fs.Root, ConfigFilename), err)
		if err != nil {
			return nil, err
		}
	}

	fs.debug(ctx, "open_repository", "complete", fs.Root, nil)
	return &retval, nil
}

// LoadConfig reads a repository configuration file from disk.
func LoadConfig(_ context.Context, loc string) (*RepositoryConfig, error) {
	var err error
	var configContents []byte
	configContents, err = os.ReadFile(path.Join(loc, ConfigFilename))
	if os.IsNotExist(err) {
		return &missingConfiguration, nil
	} else if err != nil {
		return nil, err
	}

	var config RepositoryConfig
	err = json.Unmarshal(configContents, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func writeConfig(_ context.Context, loc string, config *RepositoryConfig, mode os.FileMode) error {
	marshaled, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path.Join(loc, ConfigFilename), marshaled, mode)
}
