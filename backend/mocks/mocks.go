package mocks

import (
	"os"
	"strings"
)

// IsMockMode returns true if USE_MOCKS environment variable is set to true
func IsMockMode() bool {
	val := os.Getenv("USE_MOCKS")
	return strings.ToLower(val) == "true" || val == "1"
}
