package cloudant

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func TestBulk_AsyncFlush(t *testing.T) {
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

func TestBulk_NewEditsFalse(t *testing.T) {
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

	time.Sleep(5 * time.Second) // allow primary index to update

	rows, err := database.All(NewAllDocsQuery().Build())
	foundRevs := map[string]string{}
	for {
		row, more := <-rows
		if more {
			if r, ok := myRevs[row.ID]; ok && r == row.Value.Rev {
				foundRevs[row.ID] = row.Value.Rev
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

func TestBulk_PeriodicFlush(t *testing.T) {
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
