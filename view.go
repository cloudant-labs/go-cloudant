package cloudant

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/url"
	"strings"
)

// View contains results of Cloudant view
type View struct {
	Rows  []ViewRow `json:"rows"`
	Error string    `json:"error"`
}

// ViewRow contains one row from Cloudant view
type ViewRow struct {
	ID    string      `json:"id"`
	Key   string      `json:"key"`
	Value string      `json:"value"`
	Doc   interface{} `json:"doc"`
}

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

// List returns a channel in which AllRow types can be received.
func (d *Database) List(params url.Values) (<-chan *AllRow, error) {
	verb := "GET"
	var body []byte
	var err error

	if keysParam, ok := params["keys"]; ok {
		// If we're given a "Keys" argument, we're better off with a POST
		var keys []string
		_ = json.Unmarshal([]byte(keysParam[0]), &keys)
		body, err = json.Marshal(map[string][]string{"keys": keys})
		if err != nil {
			return nil, err
		}
		verb = "POST"
		delete(params, "keys")
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

// View allows querying views
func (d *Database) View(designName, viewName string, params url.Values, target interface{}) error {
	pathStr := "/_design/" + designName + "/_view/" + viewName

	urlStr, err := Endpoint(*d.URL, pathStr, params)
	if err != nil {
		return err
	}

	job, err := d.client.request("GET", urlStr, nil)
	defer job.Close()

	err = expectedReturnCodes(job, 200)
	if err != nil {
		return err
	}

	return json.NewDecoder(job.response.Body).Decode(&target)
}

// ViewRaw allows querying views with arbitrary output such as when using reduce
func (d *Database) ViewRaw(designName, viewName string, params url.Values) ([]byte, error) {
	pathStr := "/_design/" + designName + "/_view/" + viewName

	urlStr, err := Endpoint(*d.URL, pathStr, params)
	if err != nil {
		return nil, err
	}

	job, err := d.client.request("GET", urlStr, nil)
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
