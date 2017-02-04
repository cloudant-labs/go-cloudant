package cloudant

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type AllDocsQuery struct {
	Limit    int
	StartKey string
	EndKey   string
}

type AllDocRow struct {
	Id    string      `json:"id"`
	Value AllDocValue `json:"value"`
}

type AllDocValue struct {
	Rev string `json:"rev"`
}

type Change struct {
	Id  string
	Rev string
	Seq string
}

type ChangeResults struct {
	Id      string          `json:"id"`
	Seq     string          `json:"seq"`
	Changes []ChangeChanges `json:"changes"`
}

type ChangeChanges struct {
	Rev string `json:"rev"`
}

type Database struct {
	client *CouchClient
	Name   string
	URL    *url.URL
}

type Info struct {
	IsCompactRunning bool   `json:"compact_running"`
	DataSize         int    `json:"data_size"`
	DocDelCount      int    `json:"doc_del_count"`
	DocCount         int    `json:"doc_count"`
	DiskSize         int    `json:"disk_size"`
	UpdateSeq        string `json:"update_seq"`
}

type SetResponse struct {
	Id  string `json:"id"`
	Rev string `json:"rev"`
}

// All returns a channel in which AllDocRow types can be received
func (d *Database) All() (<-chan *AllDocRow, error) {
	return d.AllWithQuery(&AllDocsQuery{})
}

// AllWithQuery returns a channel in which AllDocRow types can be received
func (d *Database) AllWithQuery(options *AllDocsQuery) (<-chan *AllDocRow, error) {
	allDocsURL, err := url.Parse(d.URL.String())
	if err != nil {
		return nil, err
	}

	allDocsURL.Path = path.Join(allDocsURL.Path, "_all_docs")

	q := allDocsURL.Query()
	if options.Limit > 0 {
		q.Add("limit", strconv.Itoa(options.Limit))
	}
	if options.StartKey != "" {
		q.Add("startkey", options.StartKey)
	}
	if options.EndKey != "" {
		q.Add("endkey", options.EndKey)
	}

	allDocsURL.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", allDocsURL.String(), nil)

	job := CreateJob(req)
	d.client.Execute(job)

	results := make(chan *AllDocRow, 1000)

	go func(job *Job, results chan<- *AllDocRow) {
		defer job.Close()
		job.Wait()

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
				var result = new(AllDocRow)

				err := json.Unmarshal([]byte(lineStr), result)
				if err == nil {
					results <- result
				}
			}
		}
	}(job, results)

	return results, nil
}

// Bulk returns a new bulk document uploader
func (d *Database) Bulk(batchSize, concurrency int) *Uploader {
	return newUploader(d, batchSize, concurrency)
}

// Changes returns a channel in which Change types can be received
func (d *Database) Changes() (<-chan *Change, error) {
	req, err := http.NewRequest("GET", d.URL.String()+"/_changes", nil)
	if err != nil {
		return nil, err
	}

	job := CreateJob(req)
	d.client.Execute(job)

	changes := make(chan *Change, 1000)

	go func(job *Job, changes chan<- *Change) {
		defer job.Close()
		defer close(changes)

		job.Wait()

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
				var change = new(ChangeResults)

				err := json.Unmarshal([]byte(lineStr), change)
				if err == nil && len(change.Changes) == 1 {
					changes <- &Change{
						Id:  change.Id,
						Rev: change.Changes[0].Rev,
						Seq: change.Seq,
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
func (d *Database) Get(documentId string, target interface{}) error {
	return d.GetWithRev(documentId, "", target)
}

// Get a document with a specified revision.
func (d *Database) GetWithRev(documentId, rev string, target interface{}) error {
	docURL, err := url.Parse(d.URL.String())
	if err != nil {
		return err
	}

	docURL.Path = path.Join(docURL.Path, documentId)

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
func (d *Database) Delete(documentId, rev string) error {
	docURL, err := url.Parse(d.URL.String())
	if err != nil {
		return err
	}

	docURL.Path = path.Join(docURL.Path, documentId)

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
			"failed to delete document %s, status %d", documentId, job.response.StatusCode)
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

	resp := new(SetResponse)
	err = json.NewDecoder(job.response.Body).Decode(resp)

	return resp.Rev, err
}
