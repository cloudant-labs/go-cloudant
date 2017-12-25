package cloudant

// This implements some of the replication-specific endpoints of the CouchDB API.
//
// http://docs.couchdb.org/en/2.1.1/replication/protocol.html#couchdb-replication-protocol
//

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
)

// LogHistoryRow ...
type LogHistoryRow struct {
	DocWriteFailures int    `json:"doc_write_failures,omitempty"` // Number of failed writes
	DocsRead         int    `json:"docs_read,omitempty"`          // Number of read documents
	DocsWritten      int    `json:"docs_written,omitempty"`       // Number of written documents
	EndLastSeq       int    `json:"end_last_seq,omitempty"`       // Last processed Update Sequence ID
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

// EnsureFullCommitResponse ...
type EnsureFullCommitResponse struct {
	InstanceStartTime string `json:"instance_start_time"`
	OK                bool   `json:"ok"`
}

// ReplicationLog ...
type ReplicationLog struct {
	History              []LogHistoryRow `json:"history"`
	ReplicationIDVersion int             `json:"replication_id_version"`
	SessionID            string          `json:"session_id"`
	SourceLastSeq        int             `json:"source_last_seq"`
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

// FindCommonAncestry finds the most recent shared `session_id`
// See:
// http://docs.couchdb.org/en/2.1.1/replication/protocol.html#compare-replication-logs
//
// Returns either the most recent shared session_id or an empty string and false
// if no shared ancestry is found.
//
// Not sure how the `recorded_seq` fits in -- it should be an intermediate save point
func (l *ReplicationLog) FindCommonAncestry(target *ReplicationLog) (string, bool) {
	targetHistoryLen := len(target.History)
	if l.SourceLastSeq < targetHistoryLen && l.SourceLastSeq < len(l.History) {
		if l.History[l.SourceLastSeq].SessionID == target.History[l.SourceLastSeq].SessionID {
			return l.History[l.SourceLastSeq].SessionID, true
		}
	}

	lastSharedID := ""
	for i := 0; i < len(l.History); i++ {
		if i >= targetHistoryLen {
			break
		}
		if l.History[i].SessionID == target.History[i].SessionID {
			lastSharedID = l.History[i].SessionID
		}
	}

	if lastSharedID == "" {
		return "", false
	}

	return lastSharedID, true
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

// EnsureFullCommit ...
//
// http://docs.couchdb.org/en/2.1.1/replication/protocol.html#ensure-in-commit
func (d Database) EnsureFullCommit() (*EnsureFullCommitResponse, error) {
	urlStr, err := Endpoint(*d.URL, "/_ensure_full_commit", url.Values{})
	if err != nil {
		return nil, err
	}

	job, err := d.client.request("POST", urlStr, nil)
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

	ensureFullCommitResponse := &EnsureFullCommitResponse{}
	err = json.NewDecoder(job.response.Body).Decode(ensureFullCommitResponse)
	if err != nil {
		return nil, err
	}

	return ensureFullCommitResponse, nil
}

// ReplicateTo implements the CouchDB replication algorithm from the receiver
// to `destination`. `batchSize` sets the max number of documents to bulk load
// to the destination, and also determines the seq_interval for the source
// changes feed. The `concurrency` parameter sets the max number of concurrent
// batches to process.
func (d *Database) ReplicateTo(destination *Database, batchSize int, concurrency int) error {

	follower := NewFollower(d, batchSize)
	follower.heartbeat = 10000

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
		changeEvent := <-changes

		switch changeEvent.EventType {
		case ChangesHeartbeat:
		case ChangesError:
		case ChangesTerminated:
			break CHANGES
		default:
			batch = append(batch, changeEvent)
			if len(batch) >= batchSize {
				sem <- struct{}{}
				go d.handleChangesBatch(destination, batch, uploader, sem, &wg)
			}
			batch = []*ChangeEvent{}
		}
	}

	return nil
}

// BulkGet fetches many docs with their respective open revisions in one go.
//
// https://github.com/apache/couchdb-chttpd/pull/33
// https://pouchdb.com/api.html#bulk_get
func (d Database) BulkGet(body *BulkGetRequest, revs bool) (*BulkGetResponse, error) {
	params := url.Values{}
	if revs {
		params.Add("revs", "true")
	}

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

// handleChangesBatch -- main part of the replication
// Note: how does this deal with deletions
func (d Database) handleChangesBatch(destination *Database, changes []*ChangeEvent, bulker *Uploader, sem chan struct{}, wg *sync.WaitGroup) error {
	// 1. RevsDiff the batch against the destination DB
	wg.Add(1)
	defer func() { <-sem }()
	defer wg.Done() // must be executed *before* the semaphore read, hence after on the defer stack

	rd := &RevsDiffRequestBody{}
	for _, ch := range changes {
		rd.Add(ch.Meta.ID, ch.Meta.Rev)
	}

	missing, err := destination.RevsDiff(rd)
	if err != nil {
		return err
	}

	// Fetch any missing revs
	reqBody := &BulkGetRequest{}
	for ID, revs := range *missing {
		for _, rev := range revs.Missing {
			reqBody.Add(ID, rev)
		}
	}

	resp, err := d.BulkGet(reqBody, true)
	if err != nil {
		return err
	}

	// Bulk load in batches to destination
	for _, item := range resp.Results {
		for _, doc := range item.Docs {
			bulker.Upload(doc.OK)
		}
	}

	return nil
}
