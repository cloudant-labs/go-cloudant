package cloudant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type BulkDocsRequest struct {
	Docs []interface{} `json:"docs"`
}

type BulkDocsResponse struct {
	Error  string `json:"error,omitempty"`
	Id     string `json:"id"`
	Reason string `json:"reason,omitempty"`
	Rev    string `json:"rev,omitempty"`
}

type BulkJobI interface {
	getDoc() interface{}
	isPriority() bool
	done()
	Wait()
}

type BulkJob struct {
	doc      interface{}
	Error    error
	isDone   chan bool
	priority bool
	Response *BulkDocsResponse
}

func newBulkJob(doc interface{}, priority bool) *BulkJob {
	return &BulkJob{
		doc:      doc,
		Error:    nil,
		isDone:   make(chan bool, 1),
		priority: priority,
	}
}

func doneAllJobs(jobs []*BulkJob) {
	for _, j := range jobs {
		j.done()
	}
}

func errorAllJobs(jobs []*BulkJob, errMessage string) {
	for _, j := range jobs {
		j.Error = fmt.Errorf(errMessage)
	}
}

func (j *BulkJob) getDoc() interface{} { return j.doc }
func (j *BulkJob) isPriority() bool    { return j.priority }
func (j *BulkJob) done()               { j.isDone <- true }

// Wait blocks while the job is being executed.
func (j *BulkJob) Wait() { <-j.isDone }

type bulkJobFlush struct {
	async  bool
	isDone chan bool
}

func (j *bulkJobFlush) getDoc() interface{} { return nil }
func (j *bulkJobFlush) isPriority() bool    { return false }
func (j *bulkJobFlush) done()               { j.isDone <- true }
func (j *bulkJobFlush) Wait()               { <-j.isDone }

type bulkJobStop struct {
	isDone chan bool
}

func (j *bulkJobStop) getDoc() interface{} { return nil }
func (j *bulkJobStop) isPriority() bool    { return false }
func (j *bulkJobStop) done()               { j.isDone <- true }
func (j *bulkJobStop) Wait()               { <-j.isDone }

// Uploader is where Mr Smartypants live
type Uploader struct {
	concurrency int
	batchSize   int
	database    *Database
	flushTicker *time.Ticker
	uploadChan  chan BulkJobI
	workerChan  chan chan BulkJobI
	workers     []*bulkWorker
}

func newUploader(database *Database, batchSize, buffer int, flushSecs int) *Uploader {
	var flushTicker *time.Ticker
	if flushSecs > 0 {
		flushTicker = time.NewTicker(time.Duration(flushSecs) * time.Second)
	}

	uploader := Uploader{
		concurrency: database.client.workerCount,
		batchSize:   batchSize,
		database:    database,
		flushTicker: flushTicker,
		uploadChan:  make(chan BulkJobI, buffer),
		workerChan:  make(chan chan BulkJobI, database.client.workerCount),
		workers:     make([]*bulkWorker, 0),
	}

	if flushTicker != nil {
		go func() {
			for {
				select {
				case <-uploader.flushTicker.C:
					uploader.Flush()
				}
			}
		}()
	}

	uploader.start() // start workers

	return &uploader
}

// BulkUploadSimple does a one-shot synchronous bulk upload
func (u *Uploader) BulkUploadSimple(docs []interface{}) ([]BulkDocsResponse, error) {
	result, err := uploadBulkDocs(&BulkDocsRequest{docs}, u.database)
	defer result.Close()

	if err != nil || result == nil {
		LogFunc(fmt.Sprintf("bulk upload error, %s", err))
		return nil, err
	}

	if result.response == nil {
		LogFunc("bulk upload error, no response from server")
		return nil, err
	}

	if result.response.StatusCode != 201 && result.response.StatusCode != 202 {
		LogFunc(fmt.Sprintf("failed to upload bulk documents, status %d",
			result.response.StatusCode))
		return nil, err
	}

	responses := []BulkDocsResponse{}
	err = json.NewDecoder(result.response.Body).Decode(&responses)
	if err != nil {
		LogFunc(fmt.Sprintf("failed to decode /_bulk_docs response, %s", err))
		return nil, err
	}

	return responses, nil
}

// Flush blocks until all received documents have been uploaded.
func (u *Uploader) Flush() {
	job := &bulkJobFlush{isDone: make(chan bool, 1)}
	u.uploadChan <- job
	job.Wait()
}

// AsyncFlush asynchronously uploads all received documents.
func (u *Uploader) AsyncFlush() {
	job := &bulkJobFlush{async: true, isDone: make(chan bool, 1)}
	u.uploadChan <- job
	job.Wait()
}

func (u *Uploader) start() {
	// start workers
	for i := 0; i < u.concurrency; i++ {
		worker := newBulkWorker(i+1, u)
		u.workers = append(u.workers, worker)

		worker.start()
	}

	// start dispatcher
	go func() {
		for {
			job := <-u.uploadChan
			switch j := job.(type) {
			case *BulkJob:
				worker := <-u.workerChan
				worker <- j
			case *bulkJobFlush:
				flushJobs := make([]*bulkJobFlush, len(u.workers))
				for i, worker := range u.workers {
					<-u.workerChan
					flushJobs[i] = worker.flush()
				}
				if !j.async {
					for _, flushJob := range flushJobs {
						flushJob.Wait()
					}
				}
				j.done()
			case *bulkJobStop:
				stopJobs := make([]*bulkJobStop, len(u.workers))
				for i, worker := range u.workers {
					<-u.workerChan
					stopJobs[i] = worker.stop()
				}
				for _, stopJob := range stopJobs {
					stopJob.Wait()
				}
				j.done()
			}
		}
	}()
}

// Stop uploads all received documents and then terminates the upload worker(s)
func (u *Uploader) Stop() {
	if u.flushTicker != nil {
		u.flushTicker.Stop()
	}
	job := &bulkJobStop{isDone: make(chan bool, 1)}
	u.uploadChan <- job
	job.Wait()
}

// FireAndForget adds a document to the upload queue ready for processing by the upload worker(s).
func (u *Uploader) FireAndForget(doc interface{}) {
	u.uploadChan <- newBulkJob(doc, false)
}

// Upload adds a document to the upload queue ready for processing by the upload worker(s). A
// BulkJob type is returned to the client so that progress can be monitored.
func (u *Uploader) Upload(doc interface{}) *BulkJob {
	job := newBulkJob(doc, false)
	u.uploadChan <- job

	return job
}

// UploadNow adds a priority document to the upload queue ready for processing by the upload
// worker(s). Once received by a worker it triggers the upload of the entire batch (regardless of
// the current batch size). A BulkJob type is returned to the client so that progress can be
// monitored.
func (u *Uploader) UploadNow(doc interface{}) *BulkJob {
	job := newBulkJob(doc, true)
	u.uploadChan <- job

	return job
}

type bulkWorker struct {
	id       int
	jobChan  chan BulkJobI
	uploader *Uploader
}

func newBulkWorker(id int, uploader *Uploader) *bulkWorker {
	worker := &bulkWorker{
		id:       id,
		jobChan:  make(chan BulkJobI, 100),
		uploader: uploader,
	}

	return worker
}

func (w *bulkWorker) flush() *bulkJobFlush {
	job := &bulkJobFlush{isDone: make(chan bool, 1)}
	w.jobChan <- job

	return job
}

func (w *bulkWorker) start() {
	go func() {
		bulkDocs := &BulkDocsRequest{Docs: make([]interface{}, 0)}
		liveJobs := make([]*BulkJob, 0)

		for {
			w.uploader.workerChan <- w.jobChan

			job := <-w.jobChan

			switch j := job.(type) {
			case *BulkJob:
				bulkDocs.Docs = append(bulkDocs.Docs, j.getDoc())
				liveJobs = append(liveJobs, j)

				if j.isPriority() || len(bulkDocs.Docs) >= w.uploader.batchSize {
					processJobs(nil, liveJobs, bulkDocs, w.uploader)
					liveJobs = liveJobs[:0] // clear jobs
				}
			case *bulkJobFlush:
				if len(bulkDocs.Docs) > 0 {
					processJobs(j, liveJobs, bulkDocs, w.uploader)
					liveJobs = liveJobs[:0] // clear jobs
				} else {
					j.done()
				}
			case *bulkJobStop:
				if len(bulkDocs.Docs) > 0 {
					processJobs(j, liveJobs, bulkDocs, w.uploader)
				} else {
					j.done()
				}

				return
			}
		}
	}()
}

func (w *bulkWorker) stop() *bulkJobStop {
	job := &bulkJobStop{isDone: make(chan bool, 1)}
	w.jobChan <- job

	return job
}

func processJobs(parent BulkJobI, jobs []*BulkJob, req *BulkDocsRequest, uploader *Uploader) {
	result, err := uploadBulkDocs(req, uploader.database)

	go processResult(parent, jobs, result, err)

	req.Docs = req.Docs[:0] // reset
}

func processResult(parent BulkJobI, jobs []*BulkJob, result *Job, err error) {
	defer func() {
		result.Close()
		doneAllJobs(jobs)
		if parent != nil {
			parent.done()
		}
	}()

	if err != nil || result == nil {
		errMsg := fmt.Sprint("bulk upload error", err)
		LogFunc(errMsg)
		errorAllJobs(jobs, errMsg)
		return
	}

	if result.response == nil {
		errMsg := "bulk upload error, no response from server"
		LogFunc(errMsg)
		errorAllJobs(jobs, errMsg)
		return
	}

	if result.response.StatusCode != 201 && result.response.StatusCode != 202 {
		errMsg := fmt.Sprintf("failed to upload bulk documents, status %d",
			result.response.StatusCode)
		LogFunc(errMsg)
		errorAllJobs(jobs, errMsg)
		return
	}

	responses := make([]BulkDocsResponse, 0)

	err = json.NewDecoder(result.response.Body).Decode(&responses)
	if err != nil {
		errMsg := fmt.Sprintf("failed to decode /_bulk_docs response, %s", err)
		LogFunc(errMsg)
		errorAllJobs(jobs, errMsg)
		return
	}

	if len(jobs) != len(responses) {
		LogFunc("unexpected response count: %d, expected: %d", len(responses), len(jobs))
		return
	}

	for i, job := range jobs {
		job.Response = &responses[i]
		if job.Response.Error != "" {
			job.Error = fmt.Errorf("%s - %s", job.Response.Error, job.Response.Reason)
		}
	}
}

func uploadBulkDocs(bulkDocs *BulkDocsRequest, database *Database) (result *Job, err error) {
	jsonBulkDocs, err := json.Marshal(bulkDocs)
	if err != nil {
		return
	}

	b := bytes.NewReader(jsonBulkDocs)
	result, err = database.client.request("POST", database.URL.String()+"/_bulk_docs", b)

	return
}

func getByFieldName(n interface{}, field_name string) (string, bool) {
	s := reflect.ValueOf(n)

	if s.Kind() == reflect.Ptr {
		s = s.Elem()
	}

	if s.Kind() != reflect.Struct {
		return "", false
	}

	f := s.FieldByName(field_name)
	if !f.IsValid() {
		return "", false
	}

	switch f.Kind() {
	case reflect.String:
		return f.Interface().(string), true
	case reflect.Int:
		return strconv.FormatInt(f.Int(), 10), true
	default:
		return "", false
	}
}
