package cloudant

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type Database struct {
	client       *CouchClient
	DatabaseName string
	databaseURL  *url.URL
}

type DatabaseInfo struct {
	IsCompactRunning bool   `json:"compact_running"`
	DataSize         int    `json:"data_size"`
	DocDelCount      int    `json:"doc_del_count"`
	DocCount         int    `json:"doc_count"`
	DiskSize         int    `json:"disk_size"`
	UpdateSeq        string `json:"update_seq"`
}

type AllDocsRow struct {
	Id    string          `json:"id"`
	Value AllDocsDocValue `json:"value"`
}

type AllDocsDocValue struct {
	Rev string `json:"rev"`
}

func (d *Database) GetAllDocs() (chan *AllDocsRow, error) {
	req, err := http.NewRequest("GET", d.databaseURL.String()+"/_all_docs", nil)
	if err != nil {
		return nil, err
	}

	job := CreateJob(req)
	d.client.Execute(job)

	results := make(chan *AllDocsRow, 1000)

	go func(job *Job, results chan *AllDocsRow) {
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
				var result = new(AllDocsRow)

				err := json.Unmarshal([]byte(lineStr), result)
				if err == nil {
					results <- result
				}
			}
		}
	}(job, results)

	return results, nil
}

func (d *Database) GetDatabaseInfo() (info *DatabaseInfo, err error) {
	job, err := d.client.request("GET", d.databaseURL.String(), nil)
	defer job.Close()
	if err != nil {
		return
	}

	if job.response.StatusCode == 200 {
		err = json.NewDecoder(job.response.Body).Decode(&info)
	} else {
		err = fmt.Errorf("failed to get database info, status %d", job.response.StatusCode)
	}

	return
}

func (d *Database) GetDocument(docId string, target interface{}) error {
	return d.GetDocumentWithRev(docId, "", target)
}

func (d *Database) GetDocumentWithRev(docId, rev string, target interface{}) error {
	docURL, err := url.Parse(d.databaseURL.String())
	if err != nil {
		return err
	}

	docURL.Path = path.Join(docURL.Path, docId)

	if rev != "" {
		q := docURL.Query()
		q.Set("rev", rev)
	}

	job, err := d.client.request("GET", docURL.String(), nil)
	defer job.Close()
	if err != nil {
		return err
	}

	return json.NewDecoder(job.response.Body).Decode(target)
}
