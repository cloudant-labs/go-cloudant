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
	id       int
	jobChan  chan *BulkJob
	quitChan chan bool
	uploader *Uploader
}

type Uploader struct {
	concurrency int
	batchSize   int
	database    *Database
	uploadChan  chan *BulkJob
	workerChan  chan chan *BulkJob
	workers     []*bulkWorker
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
		id:       id,
		jobChan:  make(chan *BulkJob, 100),
		quitChan: make(chan bool),
		uploader: uploader,
	}

	return worker
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
		worker.stop()
	}
}

// Upload adds a document to the upload queue ready for processing by the upload worker(s)
func (u *Uploader) Upload(doc interface{}) *BulkJob {
	job := &BulkJob{
		doc:    doc,
		isDone: make(chan bool),
	}
	go func() { u.uploadChan <- job }()

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

					// reset
					liveJobs = liveJobs[:0]
					bulkDocs.Docs = bulkDocs.Docs[:0]
				}

			case <-w.quitChan:
				processJobs(liveJobs, bulkDocs, w.uploader)

				return
			}
		}
	}()
}

func (w *bulkWorker) stop() {
	go func() { w.quitChan <- true }()
}

func processJobs(jobs []*BulkJob, req *BulkDocsRequest, uploader *Uploader) {
	if len(req.Docs) == 0 {
		return
	}

	err := uploadBulkDocs(req, uploader.database)
	if err != nil {
		LogFunc("bulk upload error - %s", err)
	}

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
