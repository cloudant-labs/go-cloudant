package cloudant

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

// RevsDiffRequestBody ...
type RevsDiffRequestBody map[string][]string

func (r *RevsDiffRequestBody) Add(ID, rev string) {
	if _, ok := (*r)[ID]; ok {
		(*r)[ID] = append((*r)[ID], rev)
	} else {
		(*r)[ID] = []string{rev}
	}
}

// RevsDiffResponse ...
type RevsDiffResponse map[string]struct {
	Missing []string `json:"missing"`
}

// GetReplicationLog fetches the replication log from the local
// document given by `docID`. The `_local/` prefix will be added.
// NOTE SHOULD BE DATABASE NOT CLIENT
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

// RevsDiff ...
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

// ReplicateTo ...
func (d *Database) ReplicateTo(destination *Database, batchSize int) error {

	follower := NewFollower(d, batchSize)
	follower.heartbeat = 10000

	uploader := destination.Bulk(batchSize, 1048576, 60)

	changes, err := follower.Follow()
	if err != nil {
		return err
	}

	batch := []*ChangeEvent{}
	var wg sync.WaitGroup
	defer wg.Wait()

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
				go d.handleChangesBatch(destination, batch, uploader, &wg)
			}
			batch = []*ChangeEvent{}
		}
	}
	return nil
}

// BulkGet ...
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
func (d Database) handleChangesBatch(destination *Database, chEvs []*ChangeEvent, bulker *Uploader, wg *sync.WaitGroup) error {
	// 1. RevsDiff the batch against the destination DB
	wg.Add(1)
	rd := &RevsDiffRequestBody{}
	for _, ch := range chEvs {
		rd.Add(ch.Meta.ID, ch.Meta.Rev)
	}

	missing, err := destination.RevsDiff(rd)
	if err != nil {
		return err
	}

	// Fetch all missing bodies
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
			bulker.Upload(doc.OK) // NOTE: new_edits: false
		}
	}

	wg.Done()

	return nil
}
