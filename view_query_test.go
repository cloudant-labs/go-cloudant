package cloudant

import (
	"strings"
	"testing"
)

func TestViewQuery(t *testing.T) {

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

	viewQuery := NewViewQuery().
		Conflicts().
		DeletedConflicts().
		Descending().
		IncludeDocs().
		InclusiveEnd().
		Limit(5).
		Meta().
		R(2).
		RevsInfo().
		Skip(32)

	queryString := viewQuery.Values.Encode()

	for _, str := range expectedQueryStrings {
		if !strings.Contains(queryString, str) {
			t.Errorf("parameter encoding not found '%s'", str)
			return
		}
	}
}
