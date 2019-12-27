package cloudant

import (
	"strings"
	"testing"
)

func TestChangesQuery_Args(t *testing.T) {
	// Conflicts   bool
	// Descending  bool
	// Feed        string
	// Filter      string
	// Heartbeat   int
	// IncludeDocs bool
	// Limit       int
	// SeqInterval int
	// Since       string
	// Style       string
	// Timeout     int

	expectedQueryStrings := []string{
		"conflicts=true",
		"descending=true",
		"feed=continuous",
		"filter=_doc_ids",
		"heartbeat=5",
		"include_docs=true",
		"limit=2",
		"since=somerandomdatashouldbeSEQ",
		"style=alldocs",
		"timeout=10",
	}

	params := NewChangesQuery().
		Conflicts().
		Descending().
		Feed("continuous").
		Filter("_doc_ids").
		Heartbeat(5).
		IncludeDocs().
		Limit(2).
		Since("somerandomdatashouldbeSEQ").
		Style("alldocs").
		Timeout(10)

	queryString := params.URLValues.Encode()

	for _, str := range expectedQueryStrings {
		if !strings.Contains(queryString, str) {
			t.Errorf("parameter encoding not found '%s' in '%s'", str, queryString)
			return
		}
	}
}
