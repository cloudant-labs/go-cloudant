package cloudant

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"
)

func TestUploader_Upload(t *testing.T) {
	database := setupDatabase()

	mockBulkDocsResult := "[" +
		"{\"id\":\"doc-1\",\"error\":\"conflict\",\"reason\":\"Document update conflict.\"}," + // error: conflict
		"{\"id\":\"doc-2\",\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}," +
		"{\"id\":\"doc-3\",\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}," +
		"{\"id\":\"doc-4\",\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}," +
		"{\"id\":\"doc-5\",\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}" +
		"]"

	mockAllDocsResponse := &http.Response{
		Status:     "201 CREATED",
		StatusCode: 201,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(mockBulkDocsResult))),
	}

	setupMocks([]*http.Response{mockAllDocsResponse})

	uploader := database.Bulk(5) // batch size 5

	// upload 5 documents
	jobs := make([]*BulkJob, 5)
	for i := 0; i < 5; i++ {
		jobs[i] = uploader.Upload(TestDocument{
			Id:  fmt.Sprintf("doc-%d", i+1),
			Foo: "foobar",
			Bar: 123,
		})
	}

	for i, job := range jobs {
		job.Wait()

		if nil == job.Response {
			t.Fatal("unexpected nil job response")
		}

		if fmt.Sprintf("doc-%d", i+1) != job.Response.Id {
			t.Errorf("unexpected job %d response id %s", i+1, job.Response.Id)
		}

		if i != 0 { // success jobs 1-4
			if "1-967a00dff5e02add41819138abb3284d" != job.Response.Rev {
				t.Errorf("unexpected job %d response rev %s", i+1, job.Response.Rev)
			}
			if "" != job.Response.Error {
				t.Errorf("unexpected job %d response error %s", i+1, job.Response.Rev)
			}
			if "" != job.Response.Reason {
				t.Errorf("unexpected job %d response reason %s", i+1, job.Response.Reason)
			}
			if nil != job.Error {
				t.Errorf("unexpected job %d error %s", i+1, job.Response.Error)
			}
		} else { // failed job 0
			if "" != job.Response.Rev {
				t.Errorf("unexpected job %d response rev %s", i+1, job.Response.Rev)
			}
			if "conflict" != job.Response.Error {
				t.Errorf("unexpected job %d response error %s", i+1, job.Response.Rev)
			}
			if "Document update conflict." != job.Response.Reason {
				t.Errorf("unexpected job %d response reason %s", i+1, job.Response.Reason)
			}
			if "conflict - Document update conflict." != fmt.Sprint(job.Error) {
				t.Errorf("unexpected job %d error %s", i+1, job.Response.Error)
			}
		}
	}

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
	expectedBodyStr := "{\"docs\":[" +
		"{\"_id\":\"doc-X\",\"foo\":\"foobar\",\"bar\":123}," +
		"{\"_id\":\"doc-X\",\"foo\":\"foobar\",\"bar\":123}," +
		"{\"_id\":\"doc-X\",\"foo\":\"foobar\",\"bar\":123}," +
		"{\"_id\":\"doc-X\",\"foo\":\"foobar\",\"bar\":123}," +
		"{\"_id\":\"doc-X\",\"foo\":\"foobar\",\"bar\":123}]}"

	// remove unique doc ids as body ordering varies
	reg, _ := regexp.Compile("doc-[0-9]{1}")
	actualBodyStr := reg.ReplaceAllString(string(actualBody), "doc-X") // s/doc-[0-9]{1}/doc-X/g

	if expectedBodyStr != actualBodyStr {
		t.Errorf("unexpected request body %s", actualBodyStr)
	}
}
