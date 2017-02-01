package cloudant

func startDispatcher(client *CouchClient, workerCount int) {
	client.workers = make([]*worker, workerCount)
	client.workerChan = make(chan chan *Job)

	// create workers
	for i := 0; i < workerCount; i++ {
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
