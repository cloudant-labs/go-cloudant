package cloudant

import (
	"strings"
	"testing"
)

func TestAllDBsQuery_Args(t *testing.T) {
	// EndKey           string
	// InclusiveEnd     bool
	// Limit            int
	// Skip             int
	// StartKey         string

	expectedQueryStrings := []string{
		"inclusive_end=true",
		"limit=5",
		"startkey=%22db1%22",
		"endkey=%22db2%22",
		"skip=32",
	}

	query := NewDBsQuery().
		InclusiveEnd().
		Limit(5).
		StartKey("db1").
		Skip(32).
		EndKey("db2")

	queryString := query.URLValues.Encode()

	for _, str := range expectedQueryStrings {
		if !strings.Contains(queryString, str) {
			t.Errorf("parameter encoding not found '%s'", str)
		}
	}
}
