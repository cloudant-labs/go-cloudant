package cloudant

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestInvalidLogin(t *testing.T) {
	username := os.Getenv("COUCH_USER")
	password := "wR0ng_pa$$w0rd"

	if username == "" {
		t.Fatalf("expected env var COUCH_USER to be set")
	}

	_, err := CreateClient(username, password, "https://"+username+".cloudant.com", 5)

	if err == nil {
		t.Errorf("missing error from invalid login attempt")
	}
	if err.Error() != "failed to create session, status 401" {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestBulkAsyncFlush(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}

	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	uploader := database.Bulk(5, -1, 0)

	// upload 5 documents
	jobs := make([]*BulkJob, 5)
	for i := 0; i < 5; i++ {
		jobs[i] = uploader.Upload(cloudantDocument{
			ID:  fmt.Sprintf("doc-%d", i+1),
			Foo: "foobar",
			Bar: 123,
		})
	}

	uploader.AsyncFlush()

	for i, job := range jobs {
		job.Wait()
		if job.Response == nil {
			t.Fatal("unexpected nil job response")
		}

		if fmt.Sprintf("doc-%d", i+1) != job.Response.ID {
			t.Errorf("unexpected job %d response id %s", i+1, job.Response.ID)
		}
	}
}

func TestBulkNewEditsFalse(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}

	defer func() {
		fmt.Printf("Deleting database %s\n", database.Name)
		database.client.Delete(database.Name)
	}()

	uploader := database.Bulk(5, -1, 0)
	uploader.NewEdits = false

	// upload 5 documents
	jobs := make([]*BulkJob, 5)
	for i := 0; i < 5; i++ {
		hash, _ := dbName()

		jobs[i] = uploader.Upload(struct {
			ID  string `json:"_id"`
			Rev string `json:"_rev"`
			Foo string `json:"foo"`
		}{
			fmt.Sprintf("doc-%d", i+1),
			fmt.Sprintf("%d-%x", i+1, sha256.Sum256([]byte(hash))),
			hash,
		})
	}

	uploader.AsyncFlush()

	for _, job := range jobs {
		job.Wait()
		if job.Error != nil {
			t.Fatalf("%s", job.Error)
		}
		// new_edits=false returns no data, so can't assert based on returns
	}
}

func TestBulkAsyncFlushTwoBatches(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}

	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	uploader := database.Bulk(5, -1, 0)

	// upload 5 documents
	jobs := make([]*BulkJob, 5)
	for i := 0; i < 5; i++ {
		jobs[i] = uploader.Upload(cloudantDocument{
			ID:  fmt.Sprintf("doc-%d", i+1),
			Foo: "foobar",
			Bar: 123,
		})
	}

	uploader.AsyncFlush()

	result := []*BulkDocsResponse{}
	for i, job := range jobs {
		job.Wait()
		if job.Response == nil {
			t.Fatal("unexpected nil job response")
		}

		if job.Error != nil {
			t.Fatalf("%s", job.Error)
		}

		if fmt.Sprintf("doc-%d", i+1) != job.Response.ID {
			t.Errorf("unexpected job %d response id %s", i+1, job.Response.ID)
		}

		result = append(result, job.Response)
	}

	for i := 0; i < 5; i++ {
		foo, _ := dbName()
		jobs[i] = uploader.Upload(&struct {
			ID  string `json:"_id"`
			Rev string `json:"_rev"`
			Foo string
		}{
			result[i].ID,
			result[i].Rev,
			foo,
		})
	}

	uploader.AsyncFlush()

	for i, job := range jobs {
		job.Wait()
		if job.Response == nil {
			t.Fatal("unexpected nil job response")
		}

		if job.Error != nil {
			t.Fatalf("%s", job.Error)
		}

		if fmt.Sprintf("doc-%d", i+1) != job.Response.ID {
			t.Errorf("unexpected job %d response id %s", i+1, job.Response.ID)
		}
	}
}

func TestBulkPeriodicFlush(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	uploader := database.Bulk(10, -1, 10)

	// upload 5 documents (a partial batch)
	jobs := make([]*BulkJob, 5)
	for i := 0; i < 5; i++ {
		jobs[i] = uploader.Upload(cloudantDocument{
			ID:  fmt.Sprintf("doc-%d", i+1),
			Foo: "foobar",
			Bar: 123,
		})
	}

	// allow enough time for periodic flush to complete
	time.Sleep(30 * time.Second)

	for i, job := range jobs {
		if job.Response == nil {
			t.Fatal("unexpected nil job response")
		}

		if job.Error != nil {
			t.Fatalf("%s", job.Error)
		}

		if fmt.Sprintf("doc-%d", i+1) != job.Response.ID {
			t.Errorf("unexpected job %d response id %s", i+1, job.Response.ID)
		}
	}
}

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

func TestDatabase_AllDocArgs(t *testing.T) {
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

func TestDatabase_ChangesArgs(t *testing.T) {
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

	query := NewChangesQuery().
		Conflicts().
		Descending().
		Feed("continuous").
		Filter("_doc_ids").
		Heartbeat(5).
		IncludeDocs().
		Limit(2).
		Since("somerandomdatashouldbeSEQ").
		Style("alldocs").
		Timeout(10).
		Build()

	values, _ := query.GetQuery()
	queryString := values.Encode()

	for _, str := range expectedQueryStrings {
		if !strings.Contains(queryString, str) {
			t.Errorf("parameter encoding not found '%s' in '%s'", str, queryString)
			return
		}
	}
}

func TestDatabase_GetArgs(t *testing.T) {
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

	query := NewGetQuery().
		Attachments().
		AttEncodingInfo().
		Conflicts().
		DeletedConflicts().
		Latest().
		LocalSeq().
		Meta().
		Rev("1-bf1b7e045f2843995184f78022b3d0f5").
		Revs().
		RevsInfo().
		Build()

	values, _ := query.GetQuery()
	queryString := values.Encode()

	for _, str := range expectedQueryStrings {
		if !strings.Contains(queryString, str) {
			t.Errorf("parameter encoding not found '%s' in '%s'", str, queryString)
			return
		}
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

	rev1, err1 := database.Set(doc)
	if err1 != nil {
		t.Error("failed to create document")
		return
	}
	if !strings.HasPrefix(rev1, "1-") {
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
		rev1,
		"mydata",
		57,
	}

	rev2, err2 := database.Set(doc2)
	if err2 != nil {
		t.Error("failed to update document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	query := NewGetQuery().
		Rev(rev1).
		Build()

	err3 := database.Get("doc-new", query, doc2)
	if err3 != nil {
		t.Errorf("unexpected error %s", err3)
		return
	}

	if doc2.Rev != rev1 {
		t.Errorf("wrong revision %s", doc2.Rev)
		return
	}

	// Use the latest revision
	query = NewGetQuery().
		Rev(rev2).
		Build()

	err4 := database.Get("doc-new", query, doc2)
	if err4 != nil {
		t.Errorf("failed to fetch revision %s: %s", rev2, err4)
		return
	}

	if doc2.Rev != rev2 {
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

	rev, err := database.Set(doc)

	if err != nil {
		t.Error("failed to create document")
	}
	if !strings.HasPrefix(rev, "1-") {
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
		rev,
		"mydata",
		57,
	}

	rev, err = database.Set(doc2)
	if err != nil {
		if dberr, ok := err.(*CouchError); ok {
			t.Errorf("unexpected return code %d", dberr.StatusCode)
			return
		}
	}

	if !strings.HasPrefix(rev, "2-") {
		t.Error("got unexpected revision on update")
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

	rev, err := database.Set(doc)
	if err != nil {
		t.Error("failed to create document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	err = database.Delete("doc-new", rev)
	if err != nil {
		t.Error("failed to delete document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	err = database.Delete("doc-new", rev)
	if err == nil { // should fail
		t.Error("unexpected return code from delete")
	}
}

// TestChanges_CouchDB16 checks that we can read old-style changes feeds
// that uses a sequence ID which is an integer
func TestChanges_CouchDB16(t *testing.T) {
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
