package cloudant

import (
	"io"
	"io/ioutil"
	"net/http"
)

type Job struct {
	request  *http.Request
	response *http.Response
	error    error
	isDone   chan bool
}

func CreateJob(request *http.Request) *Job {
	job := &Job{
		request:  request,
		response: nil,
		error:    nil,
		isDone:   make(chan bool),
	}

	return job
}

func (j *Job) Close() {
	io.Copy(ioutil.Discard, j.response.Body)
	j.response.Body.Close()
}

func (j *Job) Wait() { <-j.isDone }
