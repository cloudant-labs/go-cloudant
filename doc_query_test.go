package cloudant

import (
	"strings"
	"testing"
)

func TestGetQuery_GetArgs(t *testing.T) {
	// Attachments      bool
	// AttEncodingInfo  bool
	// AttsSince        []string
	// Conflicts        bool
	// DeletedConflicts bool
	// Latest           bool
	// LocalSeq         bool
	// Meta             bool
	// OpenRevs         []string
	// Rev              string
	// Revs             bool
	// RevsInfo         bool

	expectedQueryStrings := []string{
		"attachments=true",
		"att_encoding_info=true",
		"conflicts=true",
		"deleted_conflicts=true",
		"latest=true",
		"local_seq=true",
		"meta=true",
		"rev=1-bf1b7e045f2843995184f78022b3d0f5",
		"revs=true",
		"revs_info=true",
	}

	query := NewDocQuery().
		Attachments().
		AttEncodingInfo().
		Conflicts().
		DeletedConflicts().
		Latest().
		LocalSeq().
		Meta().
		Rev("1-bf1b7e045f2843995184f78022b3d0f5").
		Revs().
		RevsInfo()

	queryString := query.Values.Encode()

	for _, str := range expectedQueryStrings {
		if !strings.Contains(queryString, str) {
			t.Errorf("parameter encoding not found '%s' in '%s'", str, queryString)
			return
		}
	}
}
