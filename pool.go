package cloudant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

// CouchError is a server error response
type CouchError struct {
	Err        string `json:"error"`
	Reason     string `json:"reason"`
	StatusCode int
}

// Error() implements the error interface
func (e *CouchError) Error() string {
	return fmt.Sprintf("%d: {%s, %s}", e.StatusCode, e.Err, e.Reason)
}

// Job wraps all requests
type Job struct {
	request    *http.Request
	response   *http.Response
	bodyBytes  []byte
	retryCount int
	error      error
	isDone     chan bool
	isLogin    bool
}

// Convenience function to check a response for errors
func expectedReturnCodes(job *Job, statusCodes ...int) error {
	for _, code := range statusCodes {
		if job.response.StatusCode == code {
			return nil
		}
	}

	dbError := &CouchError{}
	err := json.NewDecoder(job.response.Body).Decode(dbError)
	if err != nil {
		return fmt.Errorf("Failed %d", job.response.StatusCode)
	}
	dbError.StatusCode = job.response.StatusCode
	return dbError
}

// CreateJob makes a new Job from a HTTP request.
func CreateJob(request *http.Request) *Job {
	job := &Job{
		request:  request,
		response: nil,
		error:    nil,
		isDone:   make(chan bool, 1), // mark as done is non-blocking for worker
		isLogin:  false,
	}

	return job
}

// Close closes the response body reader to prevent a memory leak, even if not used
func (j *Job) Close() {
	if j.response != nil {
		io.Copy(ioutil.Discard, j.response.Body)
		j.response.Body.Close()
	}
}

// Response returns the http response
func (j *Job) Response() *http.Response {
	return j.response
}

// Mark job as done.
func (j *Job) done() { j.isDone <- true }

// Wait blocks while the job is being executed.
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
		quitChan: make(chan bool),
	}

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

			// save body for retries
			if job.retryCount == 0 && job.request.Body != nil {
				var err error
				job.bodyBytes, err = ioutil.ReadAll(job.request.Body)
				if err != nil {
					LogFunc("failed to read request body, %s", err)
				}
			}

			job.request.Body = ioutil.NopCloser(bytes.NewReader(job.bodyBytes))

			// add go-cloudant UA
			job.request.Header.Add("User-Agent", "go-cloudant/"+VERSION+"/"+runtime.Version())

			resp, err := worker.client.httpClient.Do(job.request)

			var retry bool
			if err != nil {
				LogFunc("failed to submit request, %s", err)
				retry = true
			} else {
				switch resp.StatusCode {
				case 401:
					if !job.isLogin {
						LogFunc("renewing session")
						w.client.LogIn()
						retry = true
					}
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
