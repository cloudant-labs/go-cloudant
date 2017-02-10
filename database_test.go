package cloudant

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

func setupDatabase() (database *Database) {
	client := setupClient()
	setupMocks([]*http.Response{mock412})

	database, _ = client.GetOrCreate(testDatabaseName)

	return
}

func TestDatabase_All(t *testing.T) {
	database := setupDatabase()

	mockAllDocs := "{\"total_rows\":13,\"offset\":0,\"rows\":[\n" +
		"{\"id\":\"doc1\",\"key\":\"doc1\",\"value\":{\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}},\n" +
		"{\"id\":\"doc2\",\"key\":\"doc2\",\"value\":{\"rev\":\"2-967a00dff5e02add41819138abb3284d\"}},\n" +
		"{\"id\":\"doc3\",\"key\":\"doc3\",\"value\":{\"rev\":\"3-967a00dff5e02add41819138abb3284d\"}},\n" +
		"{\"id\":\"doc4\",\"key\":\"doc4\",\"value\":{\"rev\":\"4-967a00dff5e02add41819138abb3284d\"}},\n" +
		"{\"id\":\"doc5\",\"key\":\"doc5\",\"value\":{\"rev\":\"5-967a00dff5e02add41819138abb3284d\"}}\n" +
		"]}"

	mockAllDocsResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(mockAllDocs))),
	}

	setupMocks([]*http.Response{mockAllDocsResponse})

	docs, err := database.All()
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		doc, more := <-docs
		if more {
			i += 1
			if fmt.Sprintf("doc%d", i) != doc.Id {
				t.Errorf("unexpected doc id %s", doc.Id)
			}
			if fmt.Sprintf("%d-967a00dff5e02add41819138abb3284d", i) != doc.Rev {
				t.Errorf("unexpected rev value %s", doc.Rev)
			}
		} else {
			break
		}
	}

	if 5 != i {
		t.Errorf("unexpected number of documents received %d", i)
	}

	// validate HTTP request sent to mock server
	if 1 != len(capturedJobs) {
		t.Fatalf("unexpected number of requests sent to server %d", len(capturedJobs))
	}

	job := capturedJobs[0]
	if "https://user-foo.cloudant.com/test-database-1/_all_docs" != job.request.URL.String() {
		t.Errorf("unexpected request URL %s", job.request.URL.String())
	}
	if "GET" != job.request.Method {
		t.Errorf("unexpected request method %s", job.request.Method)
	}
}

func TestDatabase_AllQ(t *testing.T) {
	database := setupDatabase()

	setupMocks([]*http.Response{mock200})

	docs, err := database.AllQ(&AllQuery{
		Limit:    123,
		StartKey: "foo1",
		EndKey:   "foo2",
	})
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-docs
		if more {
			i += 1
		} else {
			break
		}
	}

	if 0 != i {
		t.Errorf("unexpected number of documents received %d", i)
	}

	// validate HTTP request sent to mock server
	if 1 != len(capturedJobs) {
		t.Fatalf("unexpected number of requests sent to server %d", len(capturedJobs))
	}

	job := capturedJobs[0]
	if "https://user-foo.cloudant.com/test-database-1/_all_docs?endkey=foo2&limit=123&startkey=foo1" != job.request.URL.String() {
		t.Errorf("unexpected request URL %s", job.request.URL.String())
	}
	if "GET" != job.request.Method {
		t.Errorf("unexpected request method %s", job.request.Method)
	}

}

func TestDatabase_Changes(t *testing.T) {
	database := setupDatabase()

	mockChanges := "{\"results\":[\n" +
		"{\"seq\":\"1-xxxxx\",\"id\":\"doc1\",\"changes\":[{\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}]},\n" +
		"{\"seq\":\"2-xxxxx\",\"id\":\"doc2\",\"changes\":[{\"rev\":\"2-967a00dff5e02add41819138abb3284d\"}]},\n" +
		"{\"seq\":\"3-xxxxx\",\"id\":\"doc3\",\"changes\":[{\"rev\":\"3-967a00dff5e02add41819138abb3284d\"}]},\n" +
		"{\"seq\":\"4-xxxxx\",\"id\":\"doc4\",\"changes\":[{\"rev\":\"4-967a00dff5e02add41819138abb3284d\"}]},\n" +
		"{\"seq\":\"5-xxxxx\",\"id\":\"doc5\",\"changes\":[{\"rev\":\"5-967a00dff5e02add41819138abb3284d\"}]}\n" +
		"],\n" +
		"\"last_seq\":\"5-xxxxx\",\"pending\":0}"

	mockChangesResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(mockChanges))),
	}

	setupMocks([]*http.Response{mockChangesResponse})

	changes, err := database.Changes()
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		change, more := <-changes
		if more {
			i += 1
			if fmt.Sprintf("doc%d", i) != change.Id {
				t.Errorf("unexpected change id %s", change.Id)
			}
			if fmt.Sprintf("%d-xxxxx", i) != change.Seq {
				t.Errorf("unexpected change seq %s", change.Seq)
			}
			if fmt.Sprintf("%d-967a00dff5e02add41819138abb3284d", i) != change.Rev {
				t.Errorf("unexpected rev value %s", change.Rev)
			}
		} else {
			break
		}
	}

	if 5 != i {
		t.Errorf("unexpected number of changes received %d", i)
	}

	// validate HTTP request sent to mock server
	if 1 != len(capturedJobs) {
		t.Fatalf("unexpected number of requests sent to server %d", len(capturedJobs))
	}

	job := capturedJobs[0]
	if "https://user-foo.cloudant.com/test-database-1/_changes" != job.request.URL.String() {
		t.Errorf("unexpected request URL %s", job.request.URL.String())
	}
	if "GET" != job.request.Method {
		t.Errorf("unexpected request method %s", job.request.Method)
	}
}
