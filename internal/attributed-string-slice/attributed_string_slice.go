package attributedstringslice

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
)

type attributedStringSlice struct { // A "mixed-type array" in TOML.
	slice      []string
	attributes struct { // Using a struct allows for adding more attributes in the feature.
		append *bool // Nil if not set by the user
	}
}

func (ts *attributedStringSlice) UnmarshalTOML(data interface{}) error {
	iFaceSlice, ok := data.([]interface{})
	if !ok {
		return fmt.Errorf("unable to cast to interface array: %v", data)
	}

	var loadedStrings []string
	for _, x := range iFaceSlice { // Iterate over each item in the slice.
		switch val := x.(type) {
		case string: // Strings are directly appended to the slice.
			loadedStrings = append(loadedStrings, val)
		case map[string]interface{}: // The attribute struct is represented as a map.
			for k, v := range val { // Iterate over all _supported_ keys.
				switch k {
				case "append":
					boolVal, ok := v.(bool)
					if !ok {
						return fmt.Errorf("unable to cast append to bool: %v", k)
					}
					ts.attributes.append = &boolVal
				default: // Unsupported map key.
					return fmt.Errorf("unsupported key %q in map: %v", k, val)
				}
			}
		default: // Unsupported item.
			return fmt.Errorf("unsupported item in attributed string slice: %v", x)
		}
	}

	if ts.attributes.append != nil && *ts.attributes.append { // If _explicitly_ configured, append the loaded slice.
		ts.slice = append(ts.slice, loadedStrings...)
	} else { // Default: override the existing slice.
		ts.slice = loadedStrings
	}
	return nil
}

func (ts *attributedStringSlice) MarshalTOML() ([]byte, error) {
	iFaceSlice := make([]interface{}, 0, len(ts.slice))

	for _, x := range ts.slice {
		iFaceSlice = append(iFaceSlice, x)
	}

	if ts.attributes.append != nil {
		attributes := make(map[string]any)
		attributes["append"] = *ts.attributes.append
		iFaceSlice = append(iFaceSlice, attributes)
	}

	buf := new(bytes.Buffer)
	enc := toml.NewEncoder(buf)
	if err := enc.Encode(iFaceSlice); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
