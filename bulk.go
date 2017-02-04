package cloudant

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type BulkDocsRequest struct {
	Docs []interface{} `json:"docs"`
}

type BulkJob struct {
	doc    interface{}
	isDone chan bool
}

func (j *BulkJob) Wait() { <-j.isDone }

type bulkWorker struct {
	id        int
	flushChan chan *BulkJob
	jobChan   chan *BulkJob
	quitChan  chan *BulkJob
	uploader  *Uploader
}

type Uploader struct {
	concurrency int
	batchSize   int
	database    *Database
	uploadChan  chan *BulkJob
	workerChan  chan chan *BulkJob
	workers     []*bulkWorker
}

func newBulkJob(doc interface{}) *BulkJob {
	return &BulkJob{
		doc:    doc,
		isDone: make(chan bool, 1),
	}
}

func newUploader(database *Database, batchSize, concurrency int) *Uploader {
	uploader := Uploader{
		concurrency: concurrency,
		batchSize:   batchSize,
		database:    database,
		uploadChan:  make(chan *BulkJob, 100),
		workerChan:  make(chan chan *BulkJob, concurrency),
		workers:     make([]*bulkWorker, 0),
	}

	uploader.start() // start workers

	return &uploader
}

func newBulkWorker(id int, uploader *Uploader) *bulkWorker {
	worker := &bulkWorker{
		id:        id,
		flushChan: make(chan *BulkJob),
		jobChan:   make(chan *BulkJob, 100),
		quitChan:  make(chan *BulkJob),
		uploader:  uploader,
	}

	return worker
}

// Flush uploads all received documents
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
				go func() {
					worker := <-u.workerChan
					worker <- job
				}()
			}
		}
	}()
}

// Stop uploads all received documents and then terminates the upload worker(s)
func (u *Uploader) Stop() {
	for _, worker := range u.workers {
		job := worker.stop()
		job.Wait()
	}
}

// Upload adds a document to the upload queue ready for processing by the upload worker(s)
func (u *Uploader) Upload(doc interface{}) *BulkJob {
	job := newBulkJob(doc)
	go func() { u.uploadChan <- job }()

	return job
}

func (w *bulkWorker) flush() *BulkJob {
	job := newBulkJob(nil)
	go func() { w.flushChan <- job }()

	return job
}

func (w *bulkWorker) start() {
	go func() {
		bulkDocs := &BulkDocsRequest{Docs: make([]interface{}, 0)}
		liveJobs := make([]*BulkJob, 0)

		for {
			w.uploader.workerChan <- w.jobChan

			select {
			case job := <-w.jobChan:
				bulkDocs.Docs = append(bulkDocs.Docs, job.doc)
				liveJobs = append(liveJobs, job)

				if len(bulkDocs.Docs) >= w.uploader.batchSize {
					processJobs(liveJobs, bulkDocs, w.uploader)
					liveJobs = liveJobs[:0] // clear jobs
				}

			case job := <-w.flushChan:
				if len(bulkDocs.Docs) > 0 {
					processJobs(liveJobs, bulkDocs, w.uploader)
					liveJobs = liveJobs[:0] // clear jobs
				}
				job.isDone <- true // mark flush complete

			case job := <-w.quitChan:
				if len(bulkDocs.Docs) > 0 {
					processJobs(liveJobs, bulkDocs, w.uploader)
				}
				job.isDone <- true // mark flush complete

				return
			}
		}
	}()
}

func (w *bulkWorker) stop() *BulkJob {
	job := newBulkJob(nil)
	go func() { w.quitChan <- job }()

	return job
}

func processJobs(jobs []*BulkJob, req *BulkDocsRequest, uploader *Uploader) {
	if len(req.Docs) == 0 {
		return
	}

	err := uploadBulkDocs(req, uploader.database)
	if err != nil {
		LogFunc("bulk upload error - %s", err)
	}

	req.Docs = req.Docs[:0] // reset

	for _, j := range jobs {
		j.isDone <- true
	}
}

func uploadBulkDocs(bulkDocs *BulkDocsRequest, database *Database) error {
	jsonBulkDocs, err := json.Marshal(bulkDocs)
	if err != nil {
		return err
	}

	b := bytes.NewReader(jsonBulkDocs)

	job, err := database.client.request("POST", database.URL.String()+"/_bulk_docs", b)
	defer job.Close()

	if err != nil {
		return err
	}

	if job.response.StatusCode != 201 && job.response.StatusCode != 202 {
		return fmt.Errorf("failed to upload bulk documents, status %d", job.response.StatusCode)
	}

	return nil
}
