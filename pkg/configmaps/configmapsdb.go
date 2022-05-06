package configmaps

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type db struct {
	// ConfigMaps maps a configmap id to configmap metadata
	ConfigMaps map[string]ConfigMap `json:"configmaps"`
	// NameToID maps a configmap name to a configmap id
	NameToID map[string]string `json:"nameToID"`
	// IDToName maps a configmap id to a configmap name
	IDToName map[string]string `json:"idToName"`
	// lastModified is the time when the database was last modified on the file system
	lastModified time.Time
}

// loadDB loads database data into the in-memory cache if it has been modified
func (s *ConfigMapManager) loadDB() error {
	// check if the db file exists
	fileInfo, err := os.Stat(s.configMapDBPath)
	if err != nil {
		if !os.IsExist(err) {
			// If the file doesn't exist, then there's no reason to update the db cache,
			// the db cache will show no entries anyway.
			// The file will be created later on a store()
			return nil
		}
		return err
	}

	// We check if the file has been modified after the last time it was loaded into the cache.
	// If the file has been modified, then we know that our cache is not up-to-date, so we load
	// the db into the cache.
	if s.db.lastModified.Equal(fileInfo.ModTime()) {
		return nil
	}

	file, err := os.Open(s.configMapDBPath)
	if err != nil {
		return err
	}
	defer file.Close()
	if err != nil {
		return err
	}

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	unmarshalled := new(db)
	if err := json.Unmarshal(byteValue, unmarshalled); err != nil {
		return err
	}
	s.db = unmarshalled
	s.db.lastModified = fileInfo.ModTime()

	return nil
}

// getNameAndID takes a configmap's name, ID, or partial ID, and returns both its name and full ID.
func (s *ConfigMapManager) getNameAndID(nameOrID string) (name, id string, err error) {
	name, id, err = s.getExactNameAndID(nameOrID)
	if err == nil {
		return name, id, nil
	} else if errors.Cause(err) != ErrNoSuchConfigMap {
		return "", "", err
	}

	// ID prefix may have been given, iterate through all IDs.
	// ID and partial ID has a max length of 25, so we return if its greater than that.
	if len(nameOrID) > configMapIDLength {
		return "", "", errors.Wrapf(ErrNoSuchConfigMap, "no configmap with name or id %q", nameOrID)
	}
	exists := false
	var foundID, foundName string
	for id, name := range s.db.IDToName {
		if strings.HasPrefix(id, nameOrID) {
			if exists {
				return "", "", errors.Wrapf(errAmbiguous, "more than one result configmap with prefix %s", nameOrID)
			}
			exists = true
			foundID = id
			foundName = name
		}
	}

	if exists {
		return foundName, foundID, nil
	}
	return "", "", errors.Wrapf(ErrNoSuchConfigMap, "no configmap with name or id %q", nameOrID)
}

// getExactNameAndID takes a configmap's name or ID and returns both its name and full ID.
func (s *ConfigMapManager) getExactNameAndID(nameOrID string) (name, id string, err error) {
	err = s.loadDB()
	if err != nil {
		return "", "", err
	}
	if name, ok := s.db.IDToName[nameOrID]; ok {
		id := nameOrID
		return name, id, nil
	}

	if id, ok := s.db.NameToID[nameOrID]; ok {
		name := nameOrID
		return name, id, nil
	}

	return "", "", errors.Wrapf(ErrNoSuchConfigMap, "no configmap with name or id %q", nameOrID)
}

// exactConfigMapExists checks if the configmap exists, given a name or ID
// Does not match partial name or IDs
func (s *ConfigMapManager) exactConfigMapExists(nameOrID string) (bool, error) {
	_, _, err := s.getExactNameAndID(nameOrID)
	if err != nil {
		if errors.Cause(err) == ErrNoSuchConfigMap {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// lookupAll gets all configmaps stored.
func (s *ConfigMapManager) lookupAll() (map[string]ConfigMap, error) {
	err := s.loadDB()
	if err != nil {
		return nil, err
	}
	return s.db.ConfigMaps, nil
}

// lookupConfigMap returns a configmap with the given name, ID, or partial ID.
func (s *ConfigMapManager) lookupConfigMap(nameOrID string) (*ConfigMap, error) {
	err := s.loadDB()
	if err != nil {
		return nil, err
	}
	_, id, err := s.getNameAndID(nameOrID)
	if err != nil {
		return nil, err
	}
	allConfigMaps, err := s.lookupAll()
	if err != nil {
		return nil, err
	}
	if configmap, ok := allConfigMaps[id]; ok {
		return &configmap, nil
	}

	return nil, errors.Wrapf(ErrNoSuchConfigMap, "no configmap with name or id %q", nameOrID)
}

// Store creates a new configmap in the configmaps database.
// It deals with only storing metadata, not data payload.
func (s *ConfigMapManager) store(entry *ConfigMap) error {
	err := s.loadDB()
	if err != nil {
		return err
	}

	s.db.ConfigMaps[entry.ID] = *entry
	s.db.NameToID[entry.Name] = entry.ID
	s.db.IDToName[entry.ID] = entry.Name

	marshalled, err := json.MarshalIndent(s.db, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(s.configMapDBPath, marshalled, 0o600)
	if err != nil {
		return err
	}

	return nil
}

// delete deletes a configmap from the configmaps database, given a name, ID, or partial ID.
// It deals with only deleting metadata, not data payload.
func (s *ConfigMapManager) delete(nameOrID string) error {
	name, id, err := s.getNameAndID(nameOrID)
	if err != nil {
		return err
	}
	err = s.loadDB()
	if err != nil {
		return err
	}
	delete(s.db.ConfigMaps, id)
	delete(s.db.NameToID, name)
	delete(s.db.IDToName, id)
	marshalled, err := json.MarshalIndent(s.db, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(s.configMapDBPath, marshalled, 0o600)
	if err != nil {
		return err
	}
	return nil
}
