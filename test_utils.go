package cloudant

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

var capturedJobs []*Job
var mockResponses []*http.Response

// test account details
var testUsername string = "user-foo"
var testPassword string = "pa$$w0rd01"
var testDatabaseName string = "test-database-1"

type TestDocument struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}

// mock responses
var mock200 = &http.Response{
	Status:     "200 OK",
	StatusCode: 200,
	Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
}
var mock201 = &http.Response{
	Status:     "201 CREATED",
	StatusCode: 201,
	Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
}
var mock412 = &http.Response{
	Status:     "412 PRECONDITION FAILED",
	StatusCode: 412,
	Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
}

func setupClient() (client *CouchClient) {
	setupMocks([]*http.Response{mock200})

	client, _ = CreateClient(
		testUsername, testPassword, "https://"+testUsername+".cloudant.com", 1)

	return
}

func setupMocks(responses []*http.Response) {
	capturedJobs = []*Job{} // reset capture array
	mockResponses = responses

	workerFunc = func(worker *worker, job *Job) {
		//fmt.Println("captured job", job.request.URL.String())

		capturedJobs = append(capturedJobs, job)

		if len(mockResponses) == 0 {
			panic(fmt.Sprintf("unexpected request sent to server, %s",
				job.request.URL.String()))
		}

		job.response = mockResponses[0]
		mockResponses = mockResponses[1:]

		job.isDone <- true // mark as done
	}
}
