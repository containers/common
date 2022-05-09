package filedriver

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/containers/storage/pkg/lockfile"
	"github.com/pkg/errors"
)

// configMapsDataFile is the file where configMaps data/payload will be stored
var configMapsDataFile = "configmapsdata.json"

// errNoSecretData indicates that there is not data associated with an id
var errNoSecretData = errors.New("no configMap data with ID")

// errNoSecretData indicates that there is configMap data already associated with an id
var errSecretIDExists = errors.New("configMap data with ID already exists")

// Driver is the filedriver object
type Driver struct {
	// configMapsDataFilePath is the path to the configMapsfile
	configMapsDataFilePath string
	// lockfile is the filedriver lockfile
	lockfile lockfile.Locker
}

// NewDriver creates a new file driver.
// rootPath is the directory where the configMaps data file resides.
func NewDriver(rootPath string) (*Driver, error) {
	fileDriver := new(Driver)
	fileDriver.configMapsDataFilePath = filepath.Join(rootPath, configMapsDataFile)
	// the lockfile functions require that the rootPath dir is executable
	if err := os.MkdirAll(rootPath, 0o700); err != nil {
		return nil, err
	}

	lock, err := lockfile.GetLockfile(filepath.Join(rootPath, "configMapsdata.lock"))
	if err != nil {
		return nil, err
	}
	fileDriver.lockfile = lock

	return fileDriver, nil
}

// List returns all configMap IDs
func (d *Driver) List() ([]string, error) {
	d.lockfile.Lock()
	defer d.lockfile.Unlock()
	configMapData, err := d.getAllData()
	if err != nil {
		return nil, err
	}
	allID := make([]string, 0, len(configMapData))
	for k := range configMapData {
		allID = append(allID, k)
	}
	sort.Strings(allID)
	return allID, err
}

// Lookup returns the bytes associated with a configMap ID
func (d *Driver) Lookup(id string) ([]byte, error) {
	d.lockfile.Lock()
	defer d.lockfile.Unlock()

	configMapData, err := d.getAllData()
	if err != nil {
		return nil, err
	}
	if data, ok := configMapData[id]; ok {
		return data, nil
	}
	return nil, errors.Wrapf(errNoSecretData, "%s", id)
}

// Store stores the bytes associated with an ID. An error is returned if the ID arleady exists
func (d *Driver) Store(id string, data []byte) error {
	d.lockfile.Lock()
	defer d.lockfile.Unlock()

	configMapData, err := d.getAllData()
	if err != nil {
		return err
	}
	if _, ok := configMapData[id]; ok {
		return errors.Wrapf(errSecretIDExists, "%s", id)
	}
	configMapData[id] = data
	marshalled, err := json.MarshalIndent(configMapData, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(d.configMapsDataFilePath, marshalled, 0o600)
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes the configMap associated with the specified ID.  An error is returned if no matching configMap is found.
func (d *Driver) Delete(id string) error {
	d.lockfile.Lock()
	defer d.lockfile.Unlock()
	configMapData, err := d.getAllData()
	if err != nil {
		return err
	}
	if _, ok := configMapData[id]; ok {
		delete(configMapData, id)
	} else {
		return errors.Wrap(errNoSecretData, id)
	}
	marshalled, err := json.MarshalIndent(configMapData, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(d.configMapsDataFilePath, marshalled, 0o600)
	if err != nil {
		return err
	}
	return nil
}

// getAllData reads the data file and returns all data
func (d *Driver) getAllData() (map[string][]byte, error) {
	// check if the db file exists
	_, err := os.Stat(d.configMapsDataFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// the file will be created later on a store()
			return make(map[string][]byte), nil
		}
		return nil, err
	}

	file, err := os.Open(d.configMapsDataFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	configMapData := new(map[string][]byte)
	err = json.Unmarshal([]byte(byteValue), configMapData)
	if err != nil {
		return nil, err
	}
	return *configMapData, nil
}
