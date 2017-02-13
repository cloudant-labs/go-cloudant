package cloudant

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

// All requests are wrapped in a Job type.
type Job struct {
	request    *http.Request
	response   *http.Response
	retryCount int
	error      error
	isDone     chan bool
}

// Creates a new Job from a HTTP request.
func CreateJob(request *http.Request) *Job {
	job := &Job{
		request:  request,
		response: nil,
		error:    nil,
		isDone:   make(chan bool, 1), // mark as done is non-blocking for worker
	}

	return job
}

// To prevent a memory leak the response body must be closed (even when it is not used).
func (j *Job) Close() {
	if j.response != nil {
		io.Copy(ioutil.Discard, j.response.Body)
		j.response.Body.Close()
	}
}

// Mark job as done.
func (j *Job) done() { j.isDone <- true }

// Block while the job is being executed.
func (j *Job) Wait() { <-j.isDone }

type worker struct {
	id       int
	client   *CouchClient
	jobsChan chan *Job
	quitChan chan bool
}

// Create a new HTTP pool worker.
func newWorker(id int, client *CouchClient) worker {
	worker := worker{
		id:       id,
		client:   client,
		jobsChan: make(chan *Job),
		quitChan: make(chan bool)}

	return worker
}

var workerFunc func(worker *worker, job *Job) // func executed by workers

// Generates a random int within the range [min, max]
func random(min, max int) int { return rand.Intn(max-min) + min }

type CredentialsExpiredResponse struct {
	Error string `json:"error"`
}

func (w *worker) start() {
	if workerFunc == nil {
		workerFunc = func(worker *worker, job *Job) {
			LogFunc("Request (attempt: %d) %s %s", job.retryCount, job.request.Method,
				job.request.URL.String())
			resp, err := worker.client.httpClient.Do(job.request)

			var retry bool
			if err != nil {
				LogFunc("failed to submit request, %s", err)
				retry = true
			} else {
				switch resp.StatusCode {
				case 401:
					LogFunc("renewing session")
					w.client.LogIn()
					retry = true
				case 403:
					response := &CredentialsExpiredResponse{}
					err = json.NewDecoder(resp.Body).Decode(response)

					retry = false
					if err == nil && response.Error == "credentials_expired" {
						LogFunc("renewing session")
						w.client.LogIn()
						retry = true
					}
				case 429:
					retry = true
				case 500, 501, 502, 503, 504:
					retry = true
				default:
					retry = false
				}
			}

			if retry {
				if job.retryCount < w.client.retryCountMax {
					job.retryCount += 1

					go func(startDelay int) {
						time.Sleep(time.Duration(startDelay) * time.Second)
						w.client.Execute(job)
					}(random(w.client.retryDelayMin, w.client.retryDelayMax))

					return
				} else {
					LogFunc("%s %s failed, too many retries",
						job.request.Method, job.request.URL.String())
				}
			}
			job.response = resp
			job.error = err
			job.done()
		}
	}
	go func() {
		for {
			w.client.workerChan <- w.jobsChan
			select {
			case job := <-w.jobsChan:
				workerFunc(w, job)
			case <-w.quitChan:
				return
			}
		}
	}()
}

func (w *worker) stop() {
	go func() {
		w.quitChan <- true
	}()
}

func startDispatcher(client *CouchClient) {
	client.workers = make([]*worker, client.workerCount)
	client.workerChan = make(chan chan *Job, client.workerCount)

	// create workers
	for i := 0; i < client.workerCount; i++ {
		worker := newWorker(i+1, client)
		client.workers[i] = &worker
		worker.start()
	}

	go func() {
		for {
			select {
			case job := <-client.jobQueue:
				go func() {
					worker := <-client.workerChan
					worker <- job
				}()
			}
		}
	}()
}
