package cloudant

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// AllRow represents a row in the json array returned by all_docs
type AllRow struct {
	ID    string      `json:"id"`
	Value AllRowValue `json:"value"`
	Doc   interface{} `json:"doc"`
}

// AllRowValue represents a part returned by _all_docs
type AllRowValue struct {
	Rev string `json:"rev"`
}

func TestBulk_AsyncFlush(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}

	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
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

func TestBulk_NewEditsFalse(t *testing.T) {
	if travis() {
		fmt.Printf("[SKIP] TestBulk_NewEditsFalse not playing nicely with Travis")
		return
	}
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}

	defer func() {
		fmt.Printf("Deleting database %s\n", database.Name)
		database.client.Destroy(database.Name)
	}()

	uploader := database.Bulk(5, -1, 0)
	uploader.NewEdits = false

	myRevs := map[string]string{}

	// upload 5 documents
	jobs := make([]*BulkJob, 5)
	for i := 0; i < 5; i++ {
		hash, _ := dbName()

		docID := fmt.Sprintf("doc-%d", i+1)
		revID := fmt.Sprintf("%d-%x", i+1, sha256.Sum256([]byte(hash)))

		myRevs[docID] = revID

		jobs[i] = uploader.Upload(struct {
			ID  string `json:"_id"`
			Rev string `json:"_rev"`
			Foo string `json:"foo"`
		}{
			docID,
			revID,
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

	// allow primary index to update -- seems to be a particular
	// problem on travis/docker...
	time.Sleep(10 * time.Second)

	rows, err := database.List(NewViewQuery())
	foundRevs := map[string]string{}
	for {
		row, more := <-rows
		if more {
			r := new(AllRow)
			err = json.Unmarshal(row.([]byte), r)
			if err == nil {
				if rev, ok := myRevs[r.ID]; ok && rev == r.Value.Rev {
					foundRevs[r.ID] = r.Value.Rev
				}
			}
		} else {
			break
		}
	}

	if len(foundRevs) != len(myRevs) {
		t.Fatalf("Expected %d written docs, found %d", len(myRevs), len(foundRevs))
	}
}

func TestBulk_AsyncFlushTwoBatches(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}

	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
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

func TestBulk_PeriodicFlush(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
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
