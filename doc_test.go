package cloudant

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDatabase_Error4XX(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	makeDocuments(database, 10)

	doc := &cloudantDocument{}

	err = database.Get("NOTHERE", NewDocQuery(), doc)
	if err == nil {
		t.Errorf("Expected a 404 error, got nil")
		return
	}
	if dberr, ok := err.(*CouchError); ok {
		if dberr.StatusCode != 404 {
			t.Errorf("unexpected return code %d", dberr.StatusCode)
		}
	} else {
		t.Errorf("unexpected error %s", err)
	}
}

func TestDatabase_Get(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	makeDocuments(database, 10)

	doc := &cloudantDocument{}
	database.Get("doc-002", NewDocQuery(), doc)

	if doc.ID != "doc-002" {
		t.Error("failed to fetch document")
	}
}

func TestDatabase_GetWithRev(t *testing.T) {
	// Note: this is generally a bad idea, as subject to eventual consistency
	// constraints.
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}

	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	doc := &struct {
		ID  string `json:"_id"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		"mydata",
		57,
	}

	meta1, err1 := database.Insert(doc)
	if err1 != nil {
		t.Error("failed to create document")
		return
	}
	if !strings.HasPrefix(meta1.Rev, "1-") {
		t.Error("got unexpected revision on create")
		return
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	doc2 := &struct {
		ID  string `json:"_id"`
		Rev string `json:"_rev"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		meta1.Rev,
		"mydata",
		57,
	}

	meta2, err2 := database.Insert(doc2)
	if err2 != nil {
		t.Error("failed to update document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	q := NewDocQuery().
		Rev(meta1.Rev)

	err3 := database.Get("doc-new", q, doc2)
	if err3 != nil {
		t.Errorf("unexpected error %s", err3)
		return
	}

	if doc2.Rev != meta1.Rev {
		t.Errorf("wrong revision %s", doc2.Rev)
		return
	}

	// Use the latest revision
	q = NewDocQuery().
		Rev(meta2.Rev)

	err4 := database.Get("doc-new", q, doc2)
	if err4 != nil {
		t.Errorf("failed to fetch revision %s: %s", meta2.Rev, err4)
		return
	}

	if doc2.Rev != meta2.Rev {
		t.Errorf("wrong revision %s", doc2.Rev)
		return
	}
}

func TestDatabase_Set(t *testing.T) {
	// Note: this is generally a bad idea, as subject to eventual consistency
	// constraints.
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	doc := &struct {
		ID  string `json:"_id"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		"mydata",
		57,
	}

	meta, err := database.Insert(doc)

	if err != nil {
		t.Error("failed to create document")
	}
	if !strings.HasPrefix(meta.Rev, "1-") {
		t.Error("got unexpected revision on create")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	doc2 := &struct {
		ID  string `json:"_id"`
		Rev string `json:"_rev"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		meta.Rev,
		"mydata",
		57,
	}

	meta, err = database.Insert(doc2)
	if err != nil {
		if dberr, ok := err.(*CouchError); ok {
			t.Errorf("unexpected return code %d", dberr.StatusCode)
			return
		}
	}

	if !strings.HasPrefix(meta.Rev, "2-") {
		t.Error("got unexpected revision on update")
	}
}

func TestDatabase_SetNoId(t *testing.T) {
	// Note: this is generally a bad idea, as subject to eventual consistency
	// constraints.
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	doc := &struct {
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"mydata",
		57,
	}

	meta, err := database.Insert(doc)

	if err != nil {
		t.Error("failed to create document")
	}
	if !strings.HasPrefix(meta.Rev, "1-") {
		t.Error("got unexpected revision on create")
	}
}

func TestDatabase_DeleteDoc(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	doc := &struct {
		ID  string `json:"_id"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		"mydata",
		57,
	}

	meta, err := database.Insert(doc)
	if err != nil {
		t.Error("failed to create document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	err = database.Destroy("doc-new", meta.Rev)
	if err != nil {
		t.Error("failed to delete document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	err = database.Destroy("doc-new", meta.Rev)
	if err == nil { // should fail
		t.Error("unexpected return code from delete")
	}
}
