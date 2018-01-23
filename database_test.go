package cloudant

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDatabase_StaticChanges(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	makeDocuments(database, 1000)

	changes, err := database.Changes(&changesQuery{})
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-changes
		if more {
			i++
		} else {
			break
		}
	}

	if 1000 != i {
		t.Errorf("unexpected number of changes received %d", i)
	}
}

func TestDatabase_ChangesIncludeDocs(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	makeDocuments(database, 1000)
	query := NewChangesQuery().
		IncludeDocs().
		Build()

	changes, err := database.Changes(query)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		ch, more := <-changes
		if more {
			i++
		} else {
			break
		}
		if ch.Doc == nil {
			t.Error("Missing doc body")
		}
	}

	if 1000 != i {
		t.Errorf("unexpected number of changes received %d", i)
	}
}

func TestDatabase_ContinousChanges(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	makeDocuments(database, 1000)

	query := NewChangesQuery().
		Feed("continuous").
		Timeout(10).
		Build()

	changes, err := database.Changes(query)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-changes
		if more {
			i++
		} else {
			break
		}
	}

	if 1000 != i {
		t.Errorf("unexpected number of changes received %d", i)
	}
}

func TestDatabase_ChangesSeqInterval(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	makeDocuments(database, 1000)

	query := NewChangesQuery().
		SeqInterval(100).
		Build()

	changes, err := database.Changes(query)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-changes
		if more {
			i++
		} else {
			break
		}
	}

	if 1000 != i {
		t.Errorf("unexpected number of changes received %d", i)
	}
}

func TestDatabase_All(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	makeDocuments(database, 1000)

	query := NewAllDocsQuery().
		StartKey("doc-450").
		EndKey("doc-500").
		Build()

	rows, err := database.All(query)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-rows
		if more {
			i++
		} else {
			break
		}
	}

	if 51 != i {
		t.Errorf("unexpected number of rows received %d", i)
	}
}

func TestDatabase_AllDocKeys(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	makeDocuments(database, 1000)

	keys := []string{
		"doc-097",
		"doc-034",
		"doc-997",
	}

	query := NewAllDocsQuery().
		Keys(keys).
		Build()

	rows, err := database.All(query)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-rows
		if more {
			i++
		} else {
			break
		}
	}

	if 3 != i {
		t.Errorf("unexpected number of rows received %d", i)
	}
}

func TestDatabase_AllDocKey(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	makeDocuments(database, 100)

	query := NewAllDocsQuery().
		Key("doc-032").
		Build()

	rows, err := database.All(query)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-rows
		if more {
			i++
		} else {
			break
		}
	}

	if 1 != i {
		t.Errorf("unexpected number of rows received %d", i)
	}
}

func TestDatabase_Error4XX(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	makeDocuments(database, 10)

	doc := &cloudantDocument{}

	err = database.Get("NOTHERE", &getQuery{}, doc)
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
		database.client.Delete(database.Name)
	}()

	makeDocuments(database, 10)

	doc := &cloudantDocument{}
	database.Get("doc-002", &getQuery{}, doc)

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
		database.client.Delete(database.Name)
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

	meta1, err1 := database.Set(doc)
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

	meta2, err2 := database.Set(doc2)
	if err2 != nil {
		t.Error("failed to update document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	query := NewGetQuery().
		Rev(meta1.Rev).
		Build()

	err3 := database.Get("doc-new", query, doc2)
	if err3 != nil {
		t.Errorf("unexpected error %s", err3)
		return
	}

	if doc2.Rev != meta1.Rev {
		t.Errorf("wrong revision %s", doc2.Rev)
		return
	}

	// Use the latest revision
	query = NewGetQuery().
		Rev(meta2.Rev).
		Build()

	err4 := database.Get("doc-new", query, doc2)
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
		database.client.Delete(database.Name)
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

	meta, err := database.Set(doc)

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

	meta, err = database.Set(doc2)
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
		database.client.Delete(database.Name)
	}()

	doc := &struct {
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"mydata",
		57,
	}

	meta, err := database.Set(doc)

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
		database.client.Delete(database.Name)
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

	meta, err := database.Set(doc)
	if err != nil {
		t.Error("failed to create document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	err = database.Delete("doc-new", meta.Rev)
	if err != nil {
		t.Error("failed to delete document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	err = database.Delete("doc-new", meta.Rev)
	if err == nil { // should fail
		t.Error("unexpected return code from delete")
	}
}

// TestDatabase_ChangesCouchDB16 checks that we can read old-style changes feeds
// that uses a sequence ID which is an integer
func TestDatabase_ChangesCouchDB16(t *testing.T) {
	data1 := []byte(`{"seq":59,"id":"5100a7174427c7dfc1ecc5971949f201","changes":[{"rev":"1-cd6870b027e3a728bce927d4a1e0b3ab"}]}`)
	data2 := []byte(`{"seq":"59","id":"5100a7174427c7dfc1ecc5971949f201","changes":[{"rev":"1-cd6870b027e3a728bce927d4a1e0b3ab"}]}`)

	cr1 := &ChangeRow{}
	if err := json.Unmarshal(data1, cr1); err != nil {
		t.Error(err)
	}

	cr2 := &ChangeRow{}
	if err := json.Unmarshal(data2, cr2); err != nil {
		t.Error(err)
	}

	if cr1.Seq != cr2.Seq {
		t.Error("failed to parse CouchDB1.6-formatted changes data")
	}
}
