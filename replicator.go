package cloudant

// This implements some of the replication-specific endpoints of the CouchDB API.
//
// http://docs.couchdb.org/en/2.1.1/replication/protocol.html#couchdb-replication-protocol
//

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sync"
	"time"
)

type Replicator struct {
	Source      *Database
	Sink        *Database
	Concurrency int
	BatchSize   int
	Error       chan error
	Event       chan string
	Done        chan struct{}
}

// LogHistoryRow ...
type LogHistoryRow struct {
	DocWriteFailures int    `json:"doc_write_failures,omitempty"` // Number of failed writes
	DocsRead         int    `json:"docs_read,omitempty"`          // Number of read documents
	DocsWritten      int    `json:"docs_written,omitempty"`       // Number of written documents
	EndLastSeq       string `json:"end_last_seq,omitempty"`       // Last processed Update Sequence ID
	EndTime          string `json:"end_time,omitempty"`           // Replication completion timestamp in RFC 5322 format
	MissingChecked   int    `json:"missing_checked,omitempty"`    // Number of checked revisions on Source
	MissingFound     int    `json:"missing_found,omitempty"`      // Number of missing revisions found on Target
	RecordedSeq      string `json:"recorded_seq"`                 // Recorded intermediate Checkpoint. Required
	SessionID        string `json:"session_id"`                   // Unique session ID. Commonly, a random UUID value is used. Required
	StartLastSeq     string `json:"start_last_seq,omitempty"`     // Start update Sequence ID
	StartTime        string `json:"start_time,omitempty"`         // Replication start timestamp in RFC 5322 format
}

// BulkGetResponse ..
type BulkGetResponse struct {
	Results []struct {
		ID   string `json:"id"`
		Docs []struct {
			OK interface{} `json:"ok"`
		} `json:"docs"`
	} `json:"results"`
}

// BulkGetRequest ..
type BulkGetRequest struct {
	Docs []struct {
		ID  string `json:"id"`
		Rev string `json:"rev"`
	} `json:"docs"`
}

// Add adds a new id-rev pair to the request
func (b *BulkGetRequest) Add(ID, rev string) {
	b.Docs = append(b.Docs, struct {
		ID  string `json:"id"`
		Rev string `json:"rev"`
	}{
		ID,
		rev,
	})
}

// ReplicationLog ...
type ReplicationLog struct {
	History              []LogHistoryRow `json:"history"`
	ReplicationIDVersion int             `json:"replication_id_version"`
	SessionID            string          `json:"session_id"`
	SourceLastSeq        string          `json:"source_last_seq"`
}

// RevsDiffRequestBody maps ids to lists of revs
type RevsDiffRequestBody map[string][]string

// Add adds an {id, rev} pair to a _revs_diff request
func (r *RevsDiffRequestBody) Add(ID, rev string) {
	if _, ok := (*r)[ID]; ok {
		(*r)[ID] = append((*r)[ID], rev)
	} else {
		(*r)[ID] = []string{rev}
	}
}

// RevsDiffResponse maps ids to missing revs
type RevsDiffResponse map[string]struct {
	Missing []string `json:"missing"`
}

// GetReplicationLog fetches the replication log from the local
// document given by `docID`. The `_local/` prefix will be added.
//
// http://docs.couchdb.org/en/2.1.1/replication/protocol.html#retrieve-replication-logs-from-source-and-target
func (d *Database) GetReplicationLog(docID string) (*ReplicationLog, error) {
	urlStr, err := Endpoint(*d.URL, fmt.Sprintf("/_local/%s", docID), url.Values{})
	if err != nil {
		return nil, err
	}

	job, err := d.client.request("GET", urlStr, nil)
	defer func() {
		if job != nil {
			job.Close()
		}
	}()

	if err != nil {
		return nil, err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		return nil, err
	}

	replicationLog := &ReplicationLog{}
	err = json.NewDecoder(job.response.Body).Decode(replicationLog)
	if err != nil {
		return nil, err
	}

	return replicationLog, nil
}

// WriteReplicationLog writes the replication log to the local
// document given by `docID`. The `_local/` prefix will be added.
//
func (d *Database) WriteReplicationLog(docID string, newLogEntry LogHistoryRow) error {
	urlStr, err := Endpoint(*d.URL, fmt.Sprintf("/_local/%s", docID), url.Values{})
	if err != nil {
		return err
	}

	job, err := d.client.request("GET", urlStr, nil)
	defer func() {
		if job != nil {
			job.Close()
		}
	}()

	if err != nil {
		return err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		return err
	}

	replicationLog := &ReplicationLog{}
	err = json.NewDecoder(job.response.Body).Decode(replicationLog)
	if err != nil {
		return err
	}

	replicationLog.History = append([]LogHistoryRow{newLogEntry}, replicationLog.History...)

	jsonData, err := json.Marshal(replicationLog)
	if err != nil {
		return err
	}

	b := bytes.NewReader(jsonData)

	save, err := d.client.request("POST", urlStr, b)
	defer func() {
		if save != nil {
			save.Close()
		}
	}()

	if err != nil {
		return err
	}

	err = expectedReturnCodes(save, 200)
	if err != nil {
		return err
	}

	return nil
}

// FindCommonAncestry finds the most recent shared `session_id` and returns the
// corresponsing sequence id.
//
// See:
// http://docs.couchdb.org/en/2.1.1/replication/protocol.html#compare-replication-logs
//
// Returns either the sequence id corresponding to the most recent shared
// session_id or an empty string and false if no shared ancestry is found.
func (l *ReplicationLog) FindCommonAncestry(target *ReplicationLog) (string, bool) {
	// Happy path: the recorded last sequence id is for the same session
	if l.SessionID == target.SessionID {
		return l.SourceLastSeq, true
	}

	// Otherwise, examine the history array from the top (should be reverse chron)
	// and select the first item that has a corresponding session id at the target.
	for _, sourceRow := range l.History {
		for _, targetRow := range target.History {
			if targetRow.SessionID == sourceRow.SessionID {
				return sourceRow.RecordedSeq, true
			}
		}
	}

	// No shared ancestry found.
	return "", false
}

// RevsDiff when given a set of document/revision IDs, returns the subset of those
// that do not correspond to revisions stored in the database.
//
// http://docs.couchdb.org/en/2.1.1/api/database/misc.html#db-revs-diff
func (d Database) RevsDiff(body *RevsDiffRequestBody) (*RevsDiffResponse, error) {
	urlStr, err := Endpoint(*d.URL, "/_revs_diff", url.Values{})
	if err != nil {
		return nil, err
	}

	bodyJSON, err := json.Marshal(body)

	if err != nil {
		return nil, err
	}

	job, err := d.client.request("POST", urlStr, bytes.NewReader(bodyJSON))
	defer func() {
		if job != nil {
			job.Close()
		}
	}()

	if err != nil {
		return nil, err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		return nil, err
	}

	revsDiffResponse := &RevsDiffResponse{}
	err = json.NewDecoder(job.response.Body).Decode(revsDiffResponse)
	if err != nil {
		return nil, err
	}

	return revsDiffResponse, nil
}

// GenerateReplicationID uniquely identifies a replication from the receiver
// to the destination. This is potentially too simplistic.
// See http://docs.couchdb.org/en/2.1.1/replication/protocol.html#generate-replication-id
func (d *Database) GenerateReplicationID(destination *Database) string {
	str := fmt.Sprintf("%s-%s", d.URL.String(), destination.URL.String())
	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
}

// BulkGet fetches many docs with their respective open revisions in one go.
//
// https://github.com/apache/couchdb-chttpd/pull/33
// https://pouchdb.com/api.html#bulk_get
func (d Database) BulkGet(body *BulkGetRequest) (*BulkGetResponse, error) {
	params := url.Values{}
	params.Add("revs", "true")

	urlStr, err := Endpoint(*d.URL, "/_bulk_get", params)
	if err != nil {
		return nil, err
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	job, err := d.client.request("POST", urlStr, bytes.NewReader(bodyJSON))
	defer func() {
		if job != nil {
			job.done()
		}
	}()

	if err != nil {
		return nil, err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		return nil, err
	}

	resp := &BulkGetResponse{}
	err = json.NewDecoder(job.response.Body).Decode(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func uuid() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	uuid[8] = uuid[8]&^0xc0 | 0x80
	uuid[6] = uuid[6]&^0xf0 | 0x40

	return fmt.Sprintf("%x%x%x%x%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// ReplicateTo implements the CouchDB replication algorithm from the receiver
// to `destination`. `batchSize` sets the max number of documents to bulk load
// to the destination, and also determines the seq_interval for the source
// changes feed. The `concurrency` parameter sets the max number of concurrent
// batches to process.
func (r *Replicator) Replicate() error {

	replicationID := r.GenerateReplicationID()
	sourceLog, err := r.Source.GetReplicationLog(replicationID)
	destLog, err := r.Sink.GetReplicationLog(replicationID)

	follower := NewFollower(r.Source, batchSize)
	follower.heartbeat = 10000
	if since, ok := sourceLog.FindCommonAncestry(destLog); ok {
		follower.since = since
	}

	uploader := destination.Bulk(batchSize, 1048576, 60)
	uploader.NewEdits = false // upload in replicator mode to preserve source revs

	changes, err := follower.Follow()
	if err != nil {
		return err
	}

	batch := []*ChangeEvent{}

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	defer func() {
		// Ensure we drain the queue of any pending batches
		for i := 0; i < cap(sem); i++ {
			sem <- struct{}{}
		}
		// Trigger and wait for any upload workers that hold queued docs
		wg.Wait()
		uploader.Flush()
	}()

CHANGES:
	for {
		select {
		case <-r.Done:
			break CHANGES
		case changeEvent := <-changes:
			switch changeEvent.EventType {
			case ChangesHeartbeat:
			case ChangesError:
				r.Error <- changeEvent.Err
			case ChangesTerminated:
				break CHANGES
			default:
				batch = append(batch, changeEvent)
				if len(batch) >= r.BatchSize {
					sem <- struct{}{}
					go d.handleChangesBatch(
						destination,
						batch,
						uploader,
						replicationID,
						r.Error,
						sem,
						&wg,
					)
					batch = []*ChangeEvent{}
				}
			}
		}
	}

	return nil
}

// handleChangesBatch -- main part of the replication
func (d Database) handleChangesBatch(
	destination *Database,
	changes []*ChangeEvent,
	bulker *Uploader,
	replicationID string,
	errChan chan error,
	sem chan struct{},
	wg *sync.WaitGroup,
) {
	wg.Add(1)

	defer func() { <-sem }()
	defer wg.Done() // must be executed *before* the semaphore read, hence after on the defer stack

	startTime := time.Now().UTC().Format(time.RFC1123)
	sessionID, err := uuid()
	if err != nil {
		errChan <- err
		return
	}

	// 1. RevsDiff the batch against the destination DB
	rd := &RevsDiffRequestBody{}
	recordedSeq := ""
	for _, ch := range changes {
		rd.Add(ch.Meta.ID, ch.Meta.Rev)
		if ch.Seq != "" {
			recordedSeq = ch.Seq
		}
	}

	missing, err := destination.RevsDiff(rd)
	if err != nil {
		errChan <- err
		return
	}

	// 2. Fetch any missing revs. Note that CouchDB's _bulk_get isn't
	// implemented as efficiently as it could be in the clustered
	// scenario (Cloudant). We use it here to save on the HTTP overhead.
	// If running over HTTP/2 individual GET requests is potentially more
	// efficient.
	reqBody := &BulkGetRequest{}
	for ID, revs := range *missing {
		for _, rev := range revs.Missing {
			reqBody.Add(ID, rev)
		}
	}

	resp, err := d.BulkGet(reqBody)
	if err != nil {
		errChan <- err
		return
	}

	// 3. Bulk load to destination
	docs := []interface{}{}
	for _, item := range resp.Results {
		for _, doc := range item.Docs {
			docs = append(docs, doc.OK)
		}
	}

	response, err := bulker.BulkUploadSimple(docs)
	if err != nil {
		errChan <- err
		return
	}

	// Check response for errors. Individual errors not considered fatal, but
	// push onto the error channel for logging etc.
	for _, item := range response {
		if item.Error != "" {
			errChan <- fmt.Errorf(item.Error)
		}
	}

	// 4. Update replication history
	logHistoryRow := LogHistoryRow{
		StartTime:   startTime,
		EndTime:     time.Now().UTC().Format(time.RFC1123),
		RecordedSeq: recordedSeq,
		SessionID:   sessionID,
	}

	// If writing back the replication histories fail we limp on rather than
	// bailing
	err = d.WriteReplicationLog(replicationID, logHistoryRow)
	if err != nil {
		errChan <- err
	}

	err = destination.WriteReplicationLog(replicationID, logHistoryRow)
	if err != nil {
		errChan <- err
	}
}
