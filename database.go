package cloudant

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
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
	ID      string
	Rev     string
	Seq     string
	Deleted bool
	Doc     map[string]interface{} // Only present if Changes() called with include_docs=true
}

// ChangeRow represents a part returned by _changes
type ChangeRow struct {
	ID      string                 `json:"id"`
	Seq     string                 `json:"seq"` // If using CouchDB1.6, this is a number
	Changes []ChangeRowChanges     `json:"changes"`
	Deleted bool                   `json:"deleted"`
	Doc     map[string]interface{} `json:"doc"`
}

// UnmarshalJSON is here for coping with CouchDB1.6's sequence IDs being
// numbers, not strings as in Cloudant and CouchDB2.X.
//
// See https://play.golang.org/p/BytXCeHMvt
func (c *ChangeRow) UnmarshalJSON(data []byte) error {
	// Create a new type with same structure as ChangeRow but without its method set
	// to avoid an infinite `UnmarshalJSON` call stack
	type ChangeRow16 ChangeRow
	changeRow := struct {
		ChangeRow16
		Seq json.Number `json:"seq"`
	}{ChangeRow16: ChangeRow16(*c)}

	if err := json.Unmarshal(data, &changeRow); err != nil {
		return err
	}

	*c = ChangeRow(changeRow.ChangeRow16)
	c.Seq = changeRow.Seq.String()

	return nil
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
func (d *Database) All(args *allDocsQuery) (<-chan *AllRow, error) {
	verb := "GET"
	var body []byte
	var err error
	if len(args.Keys) > 0 {
		// If we're given a "Keys" argument, we're better off with a POST
		body, err = json.Marshal(map[string][]string{"keys": args.Keys})
		if err != nil {
			return nil, err
		}
		verb = "POST"
		args.Keys = nil
	}

	params, err := args.GetQuery()
	if err != nil {
		return nil, err
	}

	urlStr, err := Endpoint(*d.URL, "/_all_docs", params)
	if err != nil {
		return nil, err
	}

	job, err := d.client.request(verb, urlStr, bytes.NewReader(body))
	if err != nil {
		if job != nil {
			job.done() // close the body reader to avoid leakage
		}
		return nil, err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		job.done() // close the body reader to avoid leakage
		return nil, err
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
func (d *Database) Bulk(batchSize int, batchMaxBytes int, flushSecs int) *Uploader {
	return newUploader(d, batchSize, batchMaxBytes, bulkUploadBuffer, flushSecs)
}

// Changes returns a channel in which Change types can be received.
// See: https://console.bluemix.net/docs/services/Cloudant/api/database.html#get-changes
func (d *Database) Changes(args *changesQuery) (<-chan *Change, error) {
	verb := "GET"
	var body []byte
	var err error
	if len(args.DocIDs) > 0 {
		// If we're given a "doc_ids" argument, we're better off with a POST
		body, err = json.Marshal(map[string][]string{"doc_ids": args.DocIDs})
		if err != nil {
			return nil, err
		}
		verb = "POST"
		args.DocIDs = nil
	}

	params, err := args.GetQuery()
	if err != nil {
		return nil, err
	}

	urlStr, err := Endpoint(*d.URL, "/_changes", params)
	if err != nil {
		return nil, err
	}
	job, err := d.client.request(verb, urlStr, bytes.NewReader(body))
	if err != nil {
		job.done()
		return nil, err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		job.done()
		return nil, err
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

			if len(lineStr) > 7 && lineStr[0:7] == "{\"seq\":" {
				var change = new(ChangeRow)

				err := json.Unmarshal([]byte(lineStr), change)
				if err == nil && len(change.Changes) == 1 {
					changes <- &Change{
						ID:      change.ID,
						Rev:     change.Changes[0].Rev,
						Seq:     change.Seq,
						Doc:     change.Doc,
						Deleted: change.Deleted,
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
// See https://console.bluemix.net/docs/services/Cloudant/api/database.html#getting-database-details
func (d *Database) Info() (*Info, error) {
	job, err := d.client.request("GET", d.URL.String(), nil)
	defer job.Close()
	if err != nil {
		return nil, err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		return nil, err
	}

	info := &Info{}
	err = json.NewDecoder(job.response.Body).Decode(info)

	return info, err
}

// Get a document from the database.
// See: https://console.bluemix.net/docs/services/Cloudant/api/document.html#read
func (d *Database) Get(documentID string, args *getQuery, target interface{}) error {
	params, err := args.GetQuery()
	if err != nil {
		return err
	}
	urlStr, err := Endpoint(*d.URL, documentID, params)
	if err != nil {
		return err
	}

	job, err := d.client.request("GET", urlStr, nil)
	defer job.Close()
	if err != nil {
		return err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		return err
	}

	return json.NewDecoder(job.response.Body).Decode(target)
}

// Delete a document with a specified revision.
func (d *Database) Delete(documentID, rev string) error {
	query := url.Values{}
	query.Add("rev", rev)
	urlStr, err := Endpoint(*d.URL, documentID, query)
	if err != nil {
		return err
	}

	job, err := d.client.request("DELETE", urlStr, nil)
	defer job.Close()
	if err != nil {
		return err
	}

	return expectedReturnCodes(job, 200)
}

// Set a document. The specified type may have a json attributes '_id' and '_rev'.
// If no '_id' is given the database will generate one for you.
func (d *Database) Set(document interface{}) (*DocumentMeta, error) {
	jsonDocument, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}

	job, err := d.client.request("POST", d.URL.String(), bytes.NewReader(jsonDocument))
	defer job.Close()

	if err != nil {
		return nil, err
	}

	err = expectedReturnCodes(job, 201, 202)
	if err != nil {
		return nil, err
	}

	resp := &DocumentMeta{}
	err = json.NewDecoder(job.response.Body).Decode(resp)

	return resp, err
}
