package randid

import (
	"encoding/hex"
	"math/rand"
)

// Get generates a pseudorandom ID string of requested length.
// The string is composed of 0-9a-f characters.
//
// The string returned is random but is not guaranteed to be unique; the caller
// is expected to check if such ID is already in use and retry.
func Get(length int) string {
	b := make([]byte, length/2+length%2)
	if _, err := rand.Read(b); err != nil {
		panic(err) // This never happens.
	}

	// If the length is odd, we have to slice the result by one byte.
	return hex.EncodeToString(b)[:length]
}
