package cloudant

var workerFunc func(worker *worker, job *Job)

type worker struct {
	id       int
	client   *CouchClient
	jobsChan chan *Job
	quitChan chan bool
}

func newWorker(id int, client *CouchClient) worker {
	worker := worker{
		id:       id,
		client:   client,
		jobsChan: make(chan *Job),
		quitChan: make(chan bool)}

	return worker
}

func (w *worker) start() {
	if workerFunc == nil {
		workerFunc = func(worker *worker, job *Job) {
			LogFunc("Request: %s %s", job.request.Method, job.request.URL.String())
			resp, err := worker.client.httpClient.Do(job.request)
			job.response = resp
			job.error = err

			job.isDone <- true // mark as done
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
