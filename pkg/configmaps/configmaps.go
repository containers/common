package configmaps

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/containers/common/pkg/configmaps/filedriver"
	"github.com/containers/storage/pkg/lockfile"
	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"
)

// maxConfigMapSize is the max size for configMap data - 512kB
const maxConfigMapSize = 512000

// configMapIDLength is the character length of a configMap ID - 25
const configMapIDLength = 25

// errInvalidPath indicates that the configMaps path is invalid
var errInvalidPath = errors.New("invalid configmaps path")

// ErrNoSuchConfigMap indicates that the configMap does not exist
var ErrNoSuchConfigMap = errors.New("no such configmap")

// errConfigMapNameInUse indicates that the configMap name is already in use
var errConfigMapNameInUse = errors.New("configmap name in use")

// errInvalidConfigMapName indicates that the configMap name is invalid
var errInvalidConfigMapName = errors.New("invalid configmap name")

// errInvalidDriver indicates that the driver type is invalid
var errInvalidDriver = errors.New("invalid driver")

// errInvalidDriverOpt indicates that a driver option is invalid
var errInvalidDriverOpt = errors.New("invalid driver option")

// errAmbiguous indicates that a configMap is ambiguous
var errAmbiguous = errors.New("configmap is ambiguous")

// errDataSize indicates that the configMap data is too large or too small
var errDataSize = errors.New("configmap data must be larger than 0 and less than 512000 bytes")

// configMapsFile is the name of the file that the configMaps database will be stored in
var configMapsFile = "configmaps.json"

// configMapNameRegexp matches valid configMap names
// Allowed: 64 [a-zA-Z0-9-_.] characters, and the start and end character must be [a-zA-Z0-9]
var configMapNameRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// ConfigMapManager holds information on handling configmaps
type ConfigMapManager struct {
	// configMapDBPath is the path to the db file where configmaps are stored
	configMapDBPath string
	// lockfile is the locker for the configmap file
	lockfile lockfile.Locker
	// db is an in-memory cache of the database of configMaps
	db *db
}

// ConfigMap defines a configMap
type ConfigMap struct {
	// Name is the name of the configmap
	Name string `json:"name"`
	// ID is the unique configMap ID
	ID string `json:"id"`
	// Metadata stores other metadata on the configMap
	Metadata map[string]string `json:"metadata,omitempty"`
	// CreatedAt is when the configMap was created
	CreatedAt time.Time `json:"createdAt"`
	// Driver is the driver used to store configMap data
	Driver string `json:"driver"`
	// DriverOptions is other metadata needed to use the driver
	DriverOptions map[string]string `json:"driverOptions"`
}

// ConfigMapsDriver interfaces with the configMaps data store.
// The driver stores the actual bytes of configMap data, as opposed to
// the configMap metadata.
//
// revive does not like the name because the package is already called configmaps
//nolint:revive
type ConfigMapsDriver interface {
	// List lists all configMap ids in the configMaps data store
	List() ([]string, error)
	// Lookup gets the configMap's data bytes
	Lookup(id string) ([]byte, error)
	// Store stores the configMap's data bytes
	Store(id string, data []byte) error
	// Delete deletes a configMap's data from the driver
	Delete(id string) error
}

// NewManager creates a new configMaps manager
// rootPath is the directory where the configMaps data file resides
func NewManager(rootPath string) (*ConfigMapManager, error) {
	manager := new(ConfigMapManager)

	if !filepath.IsAbs(rootPath) {
		return nil, errors.Wrapf(errInvalidPath, "path must be absolute: %s", rootPath)
	}
	// the lockfile functions require that the rootPath dir is executable
	if err := os.MkdirAll(rootPath, 0o700); err != nil {
		return nil, err
	}

	lock, err := lockfile.GetLockfile(filepath.Join(rootPath, "configMaps.lock"))
	if err != nil {
		return nil, err
	}
	manager.lockfile = lock
	manager.configMapDBPath = filepath.Join(rootPath, configMapsFile)
	manager.db = new(db)
	manager.db.ConfigMaps = make(map[string]ConfigMap)
	manager.db.NameToID = make(map[string]string)
	manager.db.IDToName = make(map[string]string)
	return manager, nil
}

// Store takes a name, creates a configMap and stores the configMap metadata and the configMap payload.
// It returns a generated ID that is associated with the configMap.
// The max size for configMap data is 512kB.
func (s *ConfigMapManager) Store(name string, data []byte, driverType string, driverOpts map[string]string) (string, error) {
	err := validateConfigMapName(name)
	if err != nil {
		return "", err
	}

	if !(len(data) > 0 && len(data) < maxConfigMapSize) {
		return "", errDataSize
	}

	s.lockfile.Lock()
	defer s.lockfile.Unlock()

	exist, err := s.exactConfigMapExists(name)
	if err != nil {
		return "", err
	}
	if exist {
		return "", errors.Wrapf(errConfigMapNameInUse, name)
	}

	secr := new(ConfigMap)
	secr.Name = name

	for {
		newID := stringid.GenerateNonCryptoID()
		// GenerateNonCryptoID() gives 64 characters, so we truncate to correct length
		newID = newID[0:configMapIDLength]
		_, err := s.lookupConfigMap(newID)
		if err != nil {
			if errors.Cause(err) == ErrNoSuchConfigMap {
				secr.ID = newID
				break
			} else {
				return "", err
			}
		}
	}

	secr.Driver = driverType
	secr.Metadata = make(map[string]string)
	secr.CreatedAt = time.Now()
	secr.DriverOptions = driverOpts

	driver, err := getDriver(driverType, driverOpts)
	if err != nil {
		return "", err
	}
	err = driver.Store(secr.ID, data)
	if err != nil {
		return "", errors.Wrapf(err, "error creating configMap %s", name)
	}

	err = s.store(secr)
	if err != nil {
		return "", errors.Wrapf(err, "error creating configMap %s", name)
	}

	return secr.ID, nil
}

// Delete removes all configMap metadata and configMap data associated with the specified configMap.
// Delete takes a name, ID, or partial ID.
func (s *ConfigMapManager) Delete(nameOrID string) (string, error) {
	err := validateConfigMapName(nameOrID)
	if err != nil {
		return "", err
	}

	s.lockfile.Lock()
	defer s.lockfile.Unlock()

	configMap, err := s.lookupConfigMap(nameOrID)
	if err != nil {
		return "", err
	}
	configMapID := configMap.ID

	driver, err := getDriver(configMap.Driver, configMap.DriverOptions)
	if err != nil {
		return "", err
	}

	err = driver.Delete(configMapID)
	if err != nil {
		return "", errors.Wrapf(err, "error deleting configMap %s", nameOrID)
	}

	err = s.delete(configMapID)
	if err != nil {
		return "", errors.Wrapf(err, "error deleting configMap %s", nameOrID)
	}
	return configMapID, nil
}

// Lookup gives a configMap's metadata given its name, ID, or partial ID.
func (s *ConfigMapManager) Lookup(nameOrID string) (*ConfigMap, error) {
	s.lockfile.Lock()
	defer s.lockfile.Unlock()

	return s.lookupConfigMap(nameOrID)
}

// List lists all configMaps.
func (s *ConfigMapManager) List() ([]ConfigMap, error) {
	s.lockfile.Lock()
	defer s.lockfile.Unlock()

	configMaps, err := s.lookupAll()
	if err != nil {
		return nil, err
	}
	ls := make([]ConfigMap, 0, len(configMaps))
	for _, v := range configMaps {
		ls = append(ls, v)
	}
	return ls, nil
}

// LookupConfigMapData returns configMap metadata as well as configMap data in bytes.
// The configMap data can be looked up using its name, ID, or partial ID.
func (s *ConfigMapManager) LookupConfigMapData(nameOrID string) (*ConfigMap, []byte, error) {
	s.lockfile.Lock()
	defer s.lockfile.Unlock()

	configMap, err := s.lookupConfigMap(nameOrID)
	if err != nil {
		return nil, nil, err
	}
	driver, err := getDriver(configMap.Driver, configMap.DriverOptions)
	if err != nil {
		return nil, nil, err
	}
	data, err := driver.Lookup(configMap.ID)
	if err != nil {
		return nil, nil, err
	}
	return configMap, data, nil
}

// validateConfigMapName checks if the configMap name is valid.
func validateConfigMapName(name string) error {
	if !configMapNameRegexp.MatchString(name) || len(name) > 64 || strings.HasSuffix(name, "-") || strings.HasSuffix(name, ".") {
		return errors.Wrapf(errInvalidConfigMapName, "only 64 [a-zA-Z0-9-_.] characters allowed, and the start and end character must be [a-zA-Z0-9]: %s", name)
	}
	return nil
}

// getDriver creates a new driver.
func getDriver(name string, opts map[string]string) (ConfigMapsDriver, error) {
	if name == "file" {
		if path, ok := opts["path"]; ok {
			return filedriver.NewDriver(path)
		}
		return nil, errors.Wrap(errInvalidDriverOpt, "need path for filedriver")
	}
	return nil, errInvalidDriver
}
