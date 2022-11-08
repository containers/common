package report

import "strings"

// JSONFormat test CLI --format string to be a JSON request
//
//	if report.IsJSON(cmd.Flag("format").Value.String()) {
//	  ... process JSON and output
//	}
func IsJSON(s string) bool {
	return strings.TrimSpace(s) == "json"
}
