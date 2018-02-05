package cloudant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// BulkDocsRequest is the JSON body of a request to the _bulk_docs endpoint
type BulkDocsRequest struct {
	Docs     []interface{} `json:"docs"`
	NewEdits bool          `json:"new_edits"`
}

// BulkDocsResponse is the JSON body of the response from the _bulk_docs endpoint
type BulkDocsResponse struct {
	Error  string `json:"error,omitempty"`
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
	Rev    string `json:"rev,omitempty"`
}

// BulkJobI ...
type BulkJobI interface {
	getDoc() interface{}
	isPriority() bool
	done()
	Wait()
}

// BulkJob represents the state of a single document to be uploaded as part of a batch
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

func doneAllJobs(jobs *[]*BulkJob) {
	for _, j := range *jobs {
		j.done()
	}
}

func errorAllJobs(jobs *[]*BulkJob, errMessage string) {
	for _, j := range *jobs {
		j.Error = fmt.Errorf(errMessage)
	}
}

func (j *BulkJob) getDoc() interface{} { return j.doc }
func (j *BulkJob) isPriority() bool    { return j.priority }
func (j *BulkJob) done()               { j.isDone <- true }

// Wait blocks while the job is being executed.
func (j *BulkJob) Wait() { <-j.isDone }

type bulkJobFlush struct {
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
	concurrency   int
	batchSize     int
	batchMaxBytes int
	NewEdits      bool
	database      *Database
	flushTicker   *time.Ticker
	uploadChan    chan BulkJobI
	workerChan    chan chan BulkJobI
	workers       []*bulkWorker
}

func newUploader(database *Database, batchSize, batchMaxBytes, buffer int, flushSecs int) *Uploader {
	var flushTicker *time.Ticker
	if flushSecs > 0 {
		flushTicker = time.NewTicker(time.Duration(flushSecs) * time.Second)
	}

	uploader := Uploader{
		concurrency:   database.client.workerCount,
		batchSize:     batchSize,
		batchMaxBytes: batchMaxBytes,
		database:      database,
		NewEdits:      true,
		flushTicker:   flushTicker,
		uploadChan:    make(chan BulkJobI, buffer),
		workerChan:    make(chan chan BulkJobI, database.client.workerCount),
		workers:       make([]*bulkWorker, 0),
	}

	if flushTicker != nil {
		go func() {
			for {
				select {
				case <-uploader.flushTicker.C:
					uploader.AsyncFlush()
				}
			}
		}()
	}

	uploader.start() // start workers

	return &uploader
}

// BulkUploadSimple does a one-shot synchronous bulk upload
func (u *Uploader) BulkUploadSimple(docs []interface{}) ([]BulkDocsResponse, error) {
	result, err := UploadBulkDocs(&BulkDocsRequest{docs, u.NewEdits}, u.database)
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
	job := &bulkJobFlush{isDone: make(chan bool, 1)}
	u.uploadChan <- job
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
				for _, flushJob := range flushJobs {
					flushJob.Wait()
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
		liveJobs := make([]*BulkJob, 0, w.uploader.batchSize)

		if w.uploader.batchMaxBytes < 0 {
			w.uploader.batchMaxBytes = 0
		}

		bulkDocsBytes := make([]byte, 0, w.uploader.batchMaxBytes)
		initBulkDocsReq(w.uploader.NewEdits, &bulkDocsBytes)

		for {
			w.uploader.workerChan <- w.jobChan

			job := <-w.jobChan

			switch j := job.(type) {
			case *BulkJob:
				jsonDocBytes, err := json.Marshal(j.doc)
				if err != nil {
					j.Error = fmt.Errorf("invalid JSON - %s", err)
					j.done()
					break
				}

				if len(liveJobs) >= w.uploader.batchSize || (w.uploader.batchMaxBytes > 0 && len(bulkDocsBytes)+len(jsonDocBytes) > w.uploader.batchMaxBytes-2) {
					processJobs(w.uploader.NewEdits, nil, &liveJobs, &bulkDocsBytes, w.uploader)
				}

				addDoc(j, &liveJobs, &jsonDocBytes, &bulkDocsBytes)

				if j.isPriority() {
					processJobs(w.uploader.NewEdits, nil, &liveJobs, &bulkDocsBytes, w.uploader)
				}
			case *bulkJobFlush:
				processJobs(w.uploader.NewEdits, j, &liveJobs, &bulkDocsBytes, w.uploader)
			case *bulkJobStop:
				processJobs(w.uploader.NewEdits, j, &liveJobs, &bulkDocsBytes, w.uploader)
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

func initBulkDocsReq(isNewEdits bool, bulkDocsBytes *[]byte) {
	if isNewEdits {
		*bulkDocsBytes = append(*bulkDocsBytes, 123, 34, 100, 111, 99, 115, 34, 58, 91) // add '{"docs":['
	} else {
		*bulkDocsBytes = append(*bulkDocsBytes, 123, 34, 110, 101, 119, 95, 101, 100, 105, 116, 115, 34, 58, 102, 97, 108, 115, 101, 44, 34, 100, 111, 99, 115, 34, 58, 91) // add '{"new_edits":false,"docs":['
	}
}

func addDoc(job *BulkJob, liveJobs *[]*BulkJob, jsonDocBytes *[]byte, bulkDocsBytes *[]byte) {
	*liveJobs = append(*liveJobs, job)

	if len(*liveJobs) > 1 {
		*bulkDocsBytes = append(*bulkDocsBytes, 44) // add comma separator
	}

	*bulkDocsBytes = append(*bulkDocsBytes, *jsonDocBytes...)
}

func processJobs(isNewEdits bool, parent BulkJobI, jobs *[]*BulkJob, bulkDocsBytes *[]byte, uploader *Uploader) {
	defer func() {
		if parent != nil {
			parent.done()
		}
	}()

	if len(*jobs) == 0 {
		return
	}

	*bulkDocsBytes = append(*bulkDocsBytes, 93, 125) // add ']}'

	if uploader.batchMaxBytes > 0 && len(*bulkDocsBytes) > uploader.batchMaxBytes {
		errorAllJobs(jobs, "payload too large")
	} else {
		b := bytes.NewReader(*bulkDocsBytes)
		result, err := uploader.database.client.request("POST", uploader.database.URL.String()+"/_bulk_docs", b)
		processResult(jobs, result, err, isNewEdits)
	}

	*bulkDocsBytes = nil
	*jobs = nil

	initBulkDocsReq(isNewEdits, bulkDocsBytes)
}

func processResult(jobs *[]*BulkJob, result *Job, err error, isNewEdits bool) {
	defer result.Close()
	defer doneAllJobs(jobs)

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

	// Parse the responses.
	//
	// Due to a quirk in the CouchDB API, no results are returned if using
	// new_edits=false. No, that makes no sense, and yes, this is not documented
	// anywhere.
	if isNewEdits {
		responses := make([]BulkDocsResponse, 0)

		err = json.NewDecoder(result.response.Body).Decode(&responses)
		if err != nil {
			errMsg := fmt.Sprintf("failed to decode /_bulk_docs response, %s", err)
			LogFunc(errMsg)
			errorAllJobs(jobs, errMsg)
			return
		}

		if len(*jobs) != len(responses) {
			LogFunc("unexpected response count: %d, expected: %d", len(responses), len(*jobs))
			return
		}

		for i, job := range *jobs {
			job.Response = &responses[i]
			if job.Response.Error != "" {
				job.Error = fmt.Errorf("%s - %s", job.Response.Error, job.Response.Reason)
			}
		}
	}
}

// UploadBulkDocs performs a synchronous _bulk_docs POST
func UploadBulkDocs(bulkDocs *BulkDocsRequest, database *Database) (result *Job, err error) {
	jsonBulkDocs, err := json.Marshal(bulkDocs)
	if err != nil {
		return
	}

	b := bytes.NewReader(jsonBulkDocs)
	result, err = database.client.request("POST", database.URL.String()+"/_bulk_docs", b)

	return
}

func getByFieldName(n interface{}, fieldName string) (string, bool) {
	s := reflect.ValueOf(n)

	if s.Kind() == reflect.Ptr {
		s = s.Elem()
	}

	if s.Kind() != reflect.Struct {
		return "", false
	}

	f := s.FieldByName(fieldName)
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
