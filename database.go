package cloudant

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

var bulkUploadBuffer = 1000 // buffer for bulk upload channel

// AllRow represents a row in the json array returned by all_docs
type AllRow struct {
	ID    string      `json:"id"`
	Value AllRowValue `json:"value"`
	Doc   interface{} `json:"doc"`
}

// AllRowValue represents a part returned by _all_docs
type AllRowValue struct {
	Rev string `json:"rev"`
}

// Change represents a part returned by _changes
type Change struct {
	ID  string
	Rev string
	Seq string
	Doc interface{} // Only present if Changes() called with include_docs=true
}

// ChangeRow represents a part returned by _changes
type ChangeRow struct {
	ID      string             `json:"id"`
	Seq     string             `json:"seq"`
	Changes []ChangeRowChanges `json:"changes"`
	Doc     interface{}        `json:"doc"`
}

// ChangeRowChanges represents a part returned by _changes
type ChangeRowChanges struct {
	Rev string `json:"rev"`
}

// Database holds a reference to an authenticated client connection and the
// name of a remote database
type Database struct {
	client *CouchClient
	Name   string
	URL    *url.URL
}

// DocumentMeta is a CouchDB id/rev pair
type DocumentMeta struct {
	ID  string `json:"id"`
	Rev string `json:"rev"`
}

// Info represents the account meta-data
type Info struct {
	IsCompactRunning bool   `json:"compact_running"`
	DataSize         int    `json:"data_size"`
	DocDelCount      int    `json:"doc_del_count"`
	DocCount         int    `json:"doc_count"`
	DiskSize         int    `json:"disk_size"`
	UpdateSeq        string `json:"update_seq"`
}

// All returns a channel in which AllRow types can be received.
func (d *Database) All(args QueryBuilder) (<-chan *AllRow, error) {

	urlStr, err := Endpoint(*d.URL, "/_all_docs", args)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", urlStr, nil)

	job := CreateJob(req)
	d.client.Execute(job)

	job.Wait()

	if job.response.StatusCode != 200 {
		job.done()
		return nil, fmt.Errorf("failed to get database all docs, status %d",
			job.response.StatusCode)
	}

	results := make(chan *AllRow, 1000)

	go func(job *Job, results chan<- *AllRow) {
		defer job.Close()

		reader := bufio.NewReader(job.response.Body)

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				close(results)
				return
			}
			lineStr := string(line)
			lineStr = strings.TrimSpace(lineStr)      // remove whitespace
			lineStr = strings.TrimRight(lineStr, ",") // remove trailing comma

			if len(lineStr) > 7 && lineStr[0:7] == "{\"id\":\"" {
				var result = new(AllRow)

				err := json.Unmarshal([]byte(lineStr), result)
				if err == nil {
					results <- result
				}
			}
		}
	}(job, results)

	return results, nil
}

// Bulk returns a new bulk document uploader.
func (d *Database) Bulk(batchSize int) *Uploader {
	return newUploader(d, batchSize, bulkUploadBuffer)
}

// Changes returns a channel in which Change types can be received.
func (d *Database) Changes(args QueryBuilder) (<-chan *Change, error) {

	urlStr, err := Endpoint(*d.URL, "/_changes", args)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	job := CreateJob(req)
	d.client.Execute(job)

	job.Wait()

	if job.response.StatusCode != 200 {
		job.done()
		return nil, fmt.Errorf("failed to get database changes, status %d",
			job.response.StatusCode)
	}

	changes := make(chan *Change, 1000)

	go func(job *Job, changes chan<- *Change) {
		defer job.Close()
		defer close(changes)

		reader := bufio.NewReader(job.response.Body)

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}
			lineStr := string(line)
			lineStr = strings.TrimSpace(lineStr)      // remove whitespace
			lineStr = strings.TrimRight(lineStr, ",") // remove trailing comma

			if len(lineStr) > 8 && lineStr[0:8] == "{\"seq\":\"" {
				var change = new(ChangeRow)

				err := json.Unmarshal([]byte(lineStr), change)
				if err == nil && len(change.Changes) == 1 {
					changes <- &Change{
						ID:  change.ID,
						Rev: change.Changes[0].Rev,
						Seq: change.Seq,
						Doc: change.Doc,
					}
				} else {
					fmt.Println(err)
				}
			}
		}
	}(job, changes)

	return changes, nil
}

// Info returns database information.
// Attributes include document count, update seq, ...
func (d *Database) Info() (info *Info, err error) {
	job, err := d.client.request("GET", d.URL.String(), nil)
	defer job.Close()
	if err != nil {
		return
	}

	if job.response.StatusCode != 200 {
		err = fmt.Errorf("failed to get database info, status %d", job.response.StatusCode)
	}

	err = json.NewDecoder(job.response.Body).Decode(&info)

	return
}

// Get a document from the database.
// No need to specific a '_rev' as the latest revision is always returned.
func (d *Database) Get(documentID string, target interface{}) error {
	return d.GetWithRev(documentID, "", target)
}

// GetWithRev fetches a document with a specified revision.
func (d *Database) GetWithRev(documentID, rev string, target interface{}) error {
	docURL, err := url.Parse(d.URL.String())
	if err != nil {
		return err
	}

	docURL.Path = path.Join(docURL.Path, documentID)

	if rev != "" {
		q := docURL.Query()
		q.Add("rev", rev)

		docURL.RawQuery = q.Encode()
	}

	job, err := d.client.request("GET", docURL.String(), nil)
	defer job.Close()
	if err != nil {
		return err
	}

	return json.NewDecoder(job.response.Body).Decode(target)
}

// Delete a document with a specified revision.
func (d *Database) Delete(documentID, rev string) error {
	docURL, err := url.Parse(d.URL.String())
	if err != nil {
		return err
	}

	docURL.Path = path.Join(docURL.Path, documentID)

	q := docURL.Query()
	q.Add("rev", rev) // add 'rev' param

	docURL.RawQuery = q.Encode()

	job, err := d.client.request("DELETE", docURL.String(), nil)
	defer job.Close()
	if err != nil {
		return err
	}

	if job.response.StatusCode != 200 {
		return fmt.Errorf(
			"failed to delete document %s, status %d", documentID, job.response.StatusCode)
	}

	return nil
}

// Set a document. The specified type must have a json '_id' attribute.
// Be sure to also include a json '_rev' attribute if you are updating an existing document.
func (d *Database) Set(document interface{}) (string, error) {
	jsonDocument, err := json.Marshal(document)
	if err != nil {
		return "", err
	}

	b := bytes.NewReader(jsonDocument)
	job, err := d.client.request("POST", d.URL.String(), b)
	defer job.Close()

	if err != nil {
		return "", err
	}

	if job.response.StatusCode != 201 && job.response.StatusCode != 202 {
		return "", fmt.Errorf(
			"failed to delete document, status %d", job.response.StatusCode)
	}

	resp := new(DocumentMeta)
	err = json.NewDecoder(job.response.Body).Decode(resp)

	return resp.Rev, err
}
