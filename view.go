package cloudant

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
)

// ViewRow contains one row from Cloudant view
type ViewRow struct {
	ID    string      `json:"id"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
	Doc   interface{} `json:"doc"`
}

func (d *Database) viewRequest(pathStr string, q *ViewQuery) (*Job, error) {
	verb := "GET"
	var body []byte
	var err error

	if len(q.KeyValues) > 0 {
		body, err = json.Marshal(map[string][]string{"keys": q.KeyValues})
		if err != nil {
			return nil, err
		}
		verb = "POST"
	}

	urlStr, err := Endpoint(*d.URL, pathStr, q.URLValues)
	if err != nil {
		return nil, err
	}

	return d.client.request(verb, urlStr, bytes.NewReader(body))
}

// viewChannel returns a channel for a given view path in which any row interface can be received
func (d *Database) viewChannel(pathStr string, q *ViewQuery) (<-chan interface{}, error) {
	job, err := d.viewRequest(pathStr, q)
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

	results := make(chan interface{}, 1000)

	go func(job *Job, results chan<- interface{}) {
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
				results <- []byte(lineStr)
			}
		}
	}(job, results)

	return results, nil
}

// List returns a channel of all documents in which matching row types can be received.
func (d *Database) List(q *ViewQuery) (<-chan interface{}, error) {
	return d.viewChannel("/_all_docs", q)
}

// View returns a channel of view documents in which matching row types can be received.
func (d *Database) View(designName, viewName string, q *ViewQuery) (<-chan interface{}, error) {
	pathStr := "/_design/" + designName + "/_view/" + viewName
	return d.viewChannel(pathStr, q)
}

// ViewRaw allows querying views with arbitrary output such as when using reduce.
func (d *Database) ViewRaw(designName, viewName string, q *ViewQuery) ([]byte, error) {
	pathStr := "/_design/" + designName + "/_view/" + viewName
	job, err := d.viewRequest(pathStr, q)
	defer job.Close()

	err = expectedReturnCodes(job, 200)
	if err != nil {
		return nil, err
	}

	if job.response == nil {
		return nil, errors.New("Empty response")
	}

	return ioutil.ReadAll(job.response.Body)
}
