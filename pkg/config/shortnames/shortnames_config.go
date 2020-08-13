package shortnames

// TODO change this

import (
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

// PKG TODO:
// Mutexes
// More helper functions
// fullname.search implement

// EXTERNAL TODO:
// Other conf variables / defaults off << then implement here

// An alias is a short name and it's matching fullname
// to automatically pull the image from.

// DefaultMode is used when not configured
const DefaultMode = "search"

type alias struct {
	Shortname string
	Fullname  string
}

// Config is the collection of all aliased short names.
type Config struct {
	ShortnameAliasing string `toml:"shortname_aliasing,omitempty"`
	Alias             []alias
}

// nameMap is used internally in this package to merge multiple configs.
type nameMap struct {
	merged map[string][]string
}

// modeCondense accepts the alternate names for shortnames.conf shortname_aliasing
func modeCondense(mode string) int {
	switch mode {
	case "0":
		return 0
	case "1":
		return 1
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "none":
		return 0
	case "false":
		return 0
	case "read-only":
		return 1
	case "search":
		return 2
	case "prompt":
		return 3
	case "true":
		return 3
	case "always":
		return 4
	default:
		return 2
	}

}

// squash condenses the information-heavy nameMap into a Config.
func (nm nameMap) squash(conf *Config) {
	// Go maps have a non-deterministic order when iterating the keys, so
	// we dump them in a slice and sort it to enforce some order in
	// Shortnames slice.  Some consumers of c/image (e.g., CRI-O) log the
	// the configuration where a non-deterministic order could easily cause
	// confusion.
	shortnames := []string{}
	for shortname := range nm.merged {
		shortnames = append(shortnames, shortname)
	}
	sort.Strings(shortnames)

	// Place into local total version of config
	conf.Alias = []alias{}
	for _, shortname := range shortnames {
		var aliasMerge alias
		aliasMerge.Shortname = shortname
		aliasMerge.Fullname = nm.merged[shortname][0]
		conf.Alias = append(conf.Alias, aliasMerge)
	}
}

// prepend creates an alias if it doesn't exist or places it first if it does.
func (nm *nameMap) prepend(a alias) {
	var newSlice = []string{a.Fullname}
	if _, keyExists := nm.merged[a.Shortname]; keyExists {
		if index := find(nm.merged[a.Shortname], a.Fullname); index > 0 {
			nm.merged[a.Shortname] = removeIndex(nm.merged[a.Shortname], index)
			nm.merged[a.Shortname] = append(newSlice, nm.merged[a.Shortname]...)
		} else if index == -1 {
			nm.merged[a.Shortname] = append(newSlice, nm.merged[a.Shortname]...)
		}
	} else {
		nm.merged[a.Shortname] = newSlice
	}
}

// Loads a config at path into a nameMap
func (nm *nameMap) loadConfig(path string) (string, error) {

	// Load the tomlConfig. Note that `DecodeFile` will overwrite set fields.
	var conf Config
	conf.Alias = nil // important to clear the memory to prevent us from overlapping fields
	_, _ = toml.DecodeFile(path, &conf)

	//fmt.Println(conf)
	/*
		// TODO Post process shortnames, sanity checks, etc.
		if err := c.postProcess(); err != nil {
			return err
		}
	*/

	// Merge the freshly loaded shortnames on top of others.
	for _, a := range conf.Alias {
		nm.prepend(a)
	}

	nm.squash(&conf)

	if conf.ShortnameAliasing != "" {
		return conf.ShortnameAliasing, nil
	} else {
		return DefaultMode, nil
	}
}

// getAllAliases returns all aliased shortnames.
func (nm *nameMap) getAllAliases() (string, error) {
	configs, err := systemConfigs()
	if err != nil {
		return DefaultMode, err
	}

	var mode string
	for _, configPath := range configs {
		if mode, err := nm.loadConfig(configPath); err != nil {
			return mode, errors.Wrapf(err, "Issue loading %s.", configPath)
		}
	}

	return mode, nil
}

// Alias a shortname with a fullname and write out the config
func Alias(path string, shortname string, fullname string) error {
	var nm nameMap
	nm.merged = make(map[string][]string)
	if _, err := nm.loadConfig(path); err != nil {
		return errors.Wrapf(err, "Issue loading %s.", path)
	}
	var newAlias alias
	newAlias.Shortname = shortname
	newAlias.Fullname = fullname
	nm.prepend(newAlias)

	var conf Config

	nm.squash(&conf)

	conf.writeToFile(path)

	return nil
}

// Search the configs for the fullnames aliased to a shortname
// (checks if feature is enabled)
func Search(shortname string) ([]string, error) {
	var nm nameMap
	nm.merged = make(map[string][]string)
	mode, err := nm.getAllAliases()
	if err != nil {
		return nil, err
	}

	modeMux := modeCondense(mode)
	if modeMux != 0 {
		return nm.merged[shortname], nil
	}
	return nil, nil
}
