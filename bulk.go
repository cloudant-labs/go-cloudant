package cloudant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
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
	checkResult bool
	doc         interface{}
	Error       error
	isDone      chan bool
	priority    bool
	Response    *BulkDocsResponse
}

func newBulkJob(doc interface{}, priority, checkResult bool) *BulkJob {
	return &BulkJob{
		checkResult: checkResult,
		doc:         doc,
		Error:       nil,
		isDone:      make(chan bool, 1),
		priority:    priority,
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

// Block while the job is being executed.
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

type Uploader struct {
	concurrency int
	batchSize   int
	database    *Database
	uploadChan  chan BulkJobI
	workerChan  chan chan BulkJobI
	workers     []*bulkWorker
}

func newUploader(database *Database, batchSize, concurrency int) *Uploader {
	uploader := Uploader{
		concurrency: concurrency,
		batchSize:   batchSize,
		database:    database,
		uploadChan:  make(chan BulkJobI, 100),
		workerChan:  make(chan chan BulkJobI, concurrency),
		workers:     make([]*bulkWorker, 0),
	}

	uploader.start() // start workers

	return &uploader
}

// Flush uploads all received documents.
func (u *Uploader) Flush() {
	for _, worker := range u.workers {
		job := worker.flush()
		job.Wait()
	}
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
			select {
			case job := <-u.uploadChan:
				worker := <-u.workerChan
				worker <- job
			}
		}
	}()
}

// Stop uploads all received documents and then terminates the upload worker(s)
func (u *Uploader) Stop() {
	jobs := []*bulkJobStop{}
	for i := 0; i < len(u.workers); i++ {
		job := &bulkJobStop{isDone: make(chan bool, 1)}
		go func() { u.uploadChan <- job }()
		jobs = append(jobs, job)
	}

	for _, job := range jobs {
		job.Wait()
	}
}

// FireAndForget adds a document to the upload queue ready for processing by the upload worker(s).
func (u *Uploader) FireAndForget(doc interface{}) {
	u.uploadChan <- newBulkJob(doc, false, false)
}

// Upload adds a document to the upload queue ready for processing by the upload worker(s). A
// BulkJob type is returned to the client so that progress can be monitored.
func (u *Uploader) Upload(doc interface{}) *BulkJob {
	job := newBulkJob(doc, false, true)
	u.uploadChan <- job

	return job
}

// UploadNow adds a priority document to the upload queue ready for processing by the upload
// worker(s). Once received by a worker it triggers the upload of the entire batch (regardless of
// the current batch size). A BulkJob type is returned to the client so that progress can be
// monitored.
func (u *Uploader) UploadNow(doc interface{}) *BulkJob {
	job := newBulkJob(doc, true, true)
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
	go func() { w.jobChan <- job }()

	return job
}

func (w *bulkWorker) start() {
	go func() {
		bulkDocs := &BulkDocsRequest{Docs: make([]interface{}, 0)}
		liveJobs := make([]*BulkJob, 0)

		for {
			w.uploader.workerChan <- w.jobChan

			job := <-w.jobChan

			switch t := job.(type) {
			case *BulkJob:
				bulkDocs.Docs = append(bulkDocs.Docs, job.getDoc())
				liveJobs = append(liveJobs, t)

				if job.isPriority() || len(bulkDocs.Docs) >= w.uploader.batchSize {
					processJobs(liveJobs, bulkDocs, w.uploader)
					liveJobs = liveJobs[:0] // clear jobs
				}
			case *bulkJobFlush:
				if len(bulkDocs.Docs) > 0 {
					processJobs(liveJobs, bulkDocs, w.uploader)
					liveJobs = liveJobs[:0] // clear jobs
				}
				job.done()
			case *bulkJobStop:
				if len(bulkDocs.Docs) > 0 {
					processJobs(liveJobs, bulkDocs, w.uploader)
				}
				job.done()

				return
			}
		}
	}()
}

func processJobs(jobs []*BulkJob, req *BulkDocsRequest, uploader *Uploader) {
	if len(req.Docs) == 0 {
		return
	}

	result, err := uploadBulkDocs(req, uploader.database)

	go processResult(jobs, result, err)

	req.Docs = req.Docs[:0] // reset
}

func processResult(jobs []*BulkJob, result *Job, err error) {
	defer result.Close()
	defer doneAllJobs(jobs)

	if err != nil || result == nil {
		errMsg := fmt.Sprint("bulk upload error", err)
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

	for _, job := range jobs {
		if !job.checkResult {
			continue
		}

		docId, ok := getByFieldName(job.doc, "Id")
		if !ok {
			break
		}

		for _, response := range responses {
			if docId == response.Id {
				job.Response = &response
				if response.Error != "" {
					job.Error = fmt.Errorf("%s - %s", response.Error,
						response.Reason)
				}
				break
			}
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
