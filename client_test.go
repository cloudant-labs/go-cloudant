package cloudant

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

var CapturedJobs []*Job

var testUsername string = "user-foo"
var testPassword string = "pa$$w0rd01"
var testDatabaseName string = "test-database-1"

func setupClient() (client *CouchClient) {
	setupMock(&http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
	})
	client, _ = CreateClient(
		testUsername, testPassword, "https://"+testUsername+".cloudant.com", 5)

	return
}

func setupMock(mockResponse *http.Response) {
	workerFunc = func(worker *worker, job *Job) {
		CapturedJobs = []*Job{} // reset capture array
		CapturedJobs = append(CapturedJobs, job)
		job.response = mockResponse

		job.isDone <- true // mark as done
	}
}

func TestClientSessionLogIn(t *testing.T) {
	setupClient()

	if len(CapturedJobs) != 1 {
		t.Error("Unexpected request sent to server")
	}

	job := CapturedJobs[0]
	if "POST" != job.request.Method {
		t.Errorf("Unexpected request method %s", job.request.Method)
	}

	if testUsername != job.request.FormValue("name") {
		t.Errorf("Unexpected name value %s", job.request.FormValue("name"))
	}

	if testPassword != job.request.FormValue("password") {
		t.Errorf("Unexpected password value %s", job.request.FormValue("password"))
	}

	if "https://"+testUsername+".cloudant.com/_session" != job.request.URL.String() {
		t.Errorf("Unexpected request URL %s", job.request.URL.String())
	}
}

func TestClientSessionLogOut(t *testing.T) {
	client := setupClient()
	setupMock(&http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
	})

	client.LogOut()

	if len(CapturedJobs) != 1 {
		t.Error("Unexpected request sent to server")
	}

	job := CapturedJobs[0]
	if "DELETE" != job.request.Method {
		t.Errorf("Unexpected request method %s", job.request.Method)
	}

	if "https://"+testUsername+".cloudant.com/_session" != job.request.URL.String() {
		t.Errorf("Unexpected request URL %s", job.request.URL.String())
	}
}

func TestClientPing(t *testing.T) {
	client := setupClient()
	setupMock(&http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
	})

	client.Ping()

	if len(CapturedJobs) != 1 {
		t.Error("Unexpected request sent to server")
	}

	job := CapturedJobs[0]
	if "HEAD" != job.request.Method {
		t.Errorf("Unexpected request method %s", job.request.Method)
	}

	if "https://"+testUsername+".cloudant.com" != job.request.URL.String() {
		t.Errorf("Unexpected request URL %s", job.request.URL.String())
	}
}

func TestClientGetOrCreateDatabase(t *testing.T) {
	client := setupClient()
	setupMock(&http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
	})

	client.GetOrCreateDatabase(testDatabaseName)

	if len(CapturedJobs) != 1 {
		t.Error("Unexpected request sent to server")
	}

	job := CapturedJobs[0]
	if "PUT" != job.request.Method {
		t.Errorf("Unexpected request method %s", job.request.Method)
	}

	if "https://"+testUsername+".cloudant.com/"+testDatabaseName != job.request.URL.String() {
		t.Errorf("Unexpected request URL %s", job.request.URL.String())
	}
}
