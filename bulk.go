package cloudant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type BulkDocsRequest struct {
	Docs []interface{} `json:"docs"`
}

type bulkWorker struct {
	id       int
	docChan  chan *interface{}
	quitChan chan bool
	uploader *uploader
}

type uploader struct {
	concurrency int
	batchSize   int
	database    *Database
	uploadChann chan *interface{}
	workerChann chan chan *interface{}
	workers     []*bulkWorker
}

func newUploader(database *Database, batchSize, concurrency int) *uploader {
	uploader := uploader{
		concurrency: concurrency,
		batchSize:   batchSize,
		database:    database,
		uploadChann: make(chan *interface{}, 100),
		workerChann: make(chan chan *interface{}),
		workers:     make([]*bulkWorker, concurrency),
	}

	LogFunc("start workers...")
	uploader.start() // start workers

	return &uploader
}

func newBulkWorker(id int, uploader *uploader) *bulkWorker {
	worker := &bulkWorker{
		id:       id,
		docChan:  make(chan *interface{}, 100),
		quitChan: make(chan bool),
		uploader: uploader,
	}

	return worker
}

func (u *uploader) start() {
	// start workers
	for i := 0; i < u.concurrency; i++ {
		worker := newBulkWorker(i+1, u)
		u.workers[i] = worker

		LogFunc("starting...")
		worker.start()
	}

	// start dispatcher
	go func() {
		for {
			select {
			case doc := <-u.uploadChann:
				go func() {
					worker := <-u.workerChann
					worker <- doc
				}()
			}
		}
	}()
}

func (u *uploader) Upload(doc interface{}) { u.uploadChann <- &doc }

func (w *bulkWorker) start() {
	go func() {
		bulkDocs := &BulkDocsRequest{
			Docs: make([]interface{}, 0),
		}
		for {
			w.uploader.workerChann <- w.docChan
			select {
			case doc := <-w.docChan:
				if len(bulkDocs.Docs) == w.uploader.batchSize {
					err := uploadBulkDocs(bulkDocs, w.uploader.database)
					if err != nil {
						LogFunc("bulk upload error - %s", err)
					}

					bulkDocs.Docs = make([]interface{}, 0) // clear bulk docs
				}
				bulkDocs.Docs = append(bulkDocs.Docs, *doc)
			case <-w.quitChan:
				return
			}
		}
	}()
}

func uploadBulkDocs(bulkDocs *BulkDocsRequest, database *Database) error {
	jsonBulkDocs, err := json.Marshal(bulkDocs)
	if err != nil {
		return err
	}

	LogFunc("%s", jsonBulkDocs)

	b := bytes.NewReader(jsonBulkDocs)
	job, err := database.client.request("POST", database.URL.String()+"/_bulk_docs", b)
	defer job.Close()

	body, _ := ioutil.ReadAll(job.response.Body)
	LogFunc("%s", body)

	if err != nil {
		return err
	}

	if job.response.StatusCode != 201 && job.response.StatusCode != 202 {
		return fmt.Errorf("failed to upload bulk documents, status %d", job.response.StatusCode)
	}

	return nil
}
