package cloudant

import (
	"strings"
	"testing"
)

func TestAllDocsQuery_Args(t *testing.T) {
	// Conflicts        bool
	// DeletedConflicts bool
	// Descending       bool
	// EndKey           string
	// IncludeDocs      bool
	// InclusiveEnd     bool
	// Key              string
	// Keys             []string
	// Limit            int
	// Meta             bool
	// R                int
	// RevsInfo         bool
	// Skip             int
	// StartKey         string

	expectedQueryStrings := []string{
		"conflicts=true",
		"deleted_conflicts=true",
		"descending=true",
		"include_docs=true",
		"inclusive_end=true",
		"limit=5",
		"meta=true",
		"r=2",
		"revs_info=true",
		"skip=32",
	}

	query := NewAllDocsQuery().
		Conflicts().
		DeletedConflicts().
		Descending().
		IncludeDocs().
		InclusiveEnd().
		Limit(5).
		Meta().
		R(2).
		RevsInfo().
		Skip(32).
		Build()

	values, _ := query.GetQuery()
	queryString := values.Encode()

	for _, str := range expectedQueryStrings {
		if !strings.Contains(queryString, str) {
			t.Errorf("parameter encoding not found '%s'", str)
			return
		}
	}
}
