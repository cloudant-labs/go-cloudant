package cloudant

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func TestUploader_Upload(t *testing.T) {
	database := setupDatabase()

	setupMocks([]*http.Response{mock201})

	uploader := database.Bulk(5, 1) // batch size 5

	// upload 5 documents
	for i := 0; i < 5; i++ {
		uploader.Upload(TestDocument{Foo: "foobar", Bar: 123})
	}

	time.Sleep(time.Duration(1 * time.Second)) // wait for bulk jobs to hit mock server

	// validate HTTP request sent to mock server
	if 1 != len(capturedJobs) {
		t.Fatalf("unexpected number of requests sent to server %d", len(capturedJobs))
	}

	job := capturedJobs[0]
	if "https://user-foo.cloudant.com/test-database-1/_bulk_docs" != job.request.URL.String() {
		t.Errorf("unexpected request URL %s", job.request.URL.String())
	}
	if "POST" != job.request.Method {
		t.Errorf("unexpected request method %s", job.request.Method)
	}

	actualBody, _ := ioutil.ReadAll(job.request.Body)
	expectedBody := "{\"docs\":[" +
		"{\"foo\":\"foobar\",\"bar\":123}," +
		"{\"foo\":\"foobar\",\"bar\":123}," +
		"{\"foo\":\"foobar\",\"bar\":123}," +
		"{\"foo\":\"foobar\",\"bar\":123}," +
		"{\"foo\":\"foobar\",\"bar\":123}]}"
	if expectedBody != string(actualBody) {
		t.Errorf("unexpected request body %s", actualBody)
	}
}
