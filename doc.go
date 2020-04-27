package cloudant

import (
	"bytes"
	"encoding/json"
	"net/url"
)

// DocumentMeta is a CouchDB id/rev pair.
type DocumentMeta struct {
	ID  string `json:"id"`
	Rev string `json:"rev"`
}

// Get a document from the database.
// See: https://console.bluemix.net/docs/services/Cloudant/api/document.html#read
func (d *Database) Get(documentID string, q *DocQuery, target interface{}) error {
	urlStr, err := Endpoint(*d.URL, documentID, q.URLValues)
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

// Destroy a document with a specified revision.
func (d *Database) Destroy(documentID, rev string) error {
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

// InsertRaw posts raw input to Cloudant.
// Input may have json attributes '_id' and '_rev'.
// If no '_id' is given the database will generate one for you.
func (d *Database) InsertRaw(jsonDocument []byte) (*DocumentMeta, error) {
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

// UnescapedHTMLJSONMarshal marshals JSON without escaping HTML.
func UnescapedHTMLJSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

// Insert a document without escaped HTML.
func (d *Database) Insert(document interface{}) (*DocumentMeta, error) {
	// Custom marshaller with SetEscapeHTML set to false
	jsonDocument, err := UnescapedHTMLJSONMarshal(document)
	if err != nil {
		return nil, err
	}

	return d.InsertRaw(jsonDocument)
}

// InsertEscaped a document with escaped HTML.
func (d *Database) InsertEscaped(document interface{}) (*DocumentMeta, error) {
	//json.Marshal by default encodes HTML in strings
	jsonDocument, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}

	return d.InsertRaw(jsonDocument)
}
