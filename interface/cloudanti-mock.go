/**
 * Cloudant interface mock functions
 * - Provides example convenience mock functions to the main library
 * - Use directly or as an example for your own implementation
 */

package cloudanti

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/cloudant-labs/go-cloudant"
	"github.com/buger/jsonparser"
)

// CloudantContent holds content for mock Cloudant instance.
type CloudantContent struct {
	Databases map[string]DatabaseContent
}

// DatabaseContent holds mock content for mock database.
type DatabaseContent struct {
	Info  cloudant.Info
	Docs  map[string][]byte
	Views map[string][]string // map design URL, e.g. /_design/search~view/_view/versions?descending=true&include_docs=true&key=PLAN-abcd&limit=1 to list of doc IDs from docs
}

type mockClientImpl struct {
	databases map[string]DatabaseContent
}

type mockDatabaseImpl struct {
	client       *mockClientImpl
	databaseName string
}

// NewMockClient mocks the creation of a new Cloudant instance.
func NewMockClient(content CloudantContent) (Client, error) {
	if content.Databases == nil {
		content.Databases = make(map[string]DatabaseContent)
	}
	return &mockClientImpl{
		databases: content.Databases,
	}, nil
}

// Exists checks the existence of a specified mock database.
// Returns true if the database exists, else false.
func (c *mockClientImpl) Exists(databaseName string) (bool, error) {
	_, exists := c.databases[databaseName]
	return exists, nil
}

// Info returns mock database information.
// See https://console.bluemix.net/docs/services/Cloudant/api/database.html#getting-database-details
func (c *mockClientImpl) Info(databaseName string) (*cloudant.Info, error) {
	db, exists := c.databases[databaseName]
	if !exists {
		return nil, errors.New("Database does not exist")
	}
	return &db.Info, nil
}

// Use mocks pointing to a database as in NodeJS nano library.
func (c *mockClientImpl) Use(databaseName string) (Database, error) {
	return &mockDatabaseImpl{
		client:       c,
		databaseName: databaseName,
	}, nil
}

// UseOrCreate mocks pointing to a database as in NodeJS nano library but creates the database if missing.
func (c *mockClientImpl) UseOrCreate(databaseName string) (Database, error) {
	_, exists := c.databases[databaseName]
	if !exists {
		c.databases[databaseName] = DatabaseContent{
			Docs:  make(map[string][]byte),
			Views: make(map[string][]string),
		}
	}
	return &mockDatabaseImpl{
		client:       c,
		databaseName: databaseName,
	}, nil
}

// Destroy hard deletes database.
func (c *mockClientImpl) Destroy(databaseName string) error {
	delete(c.databases, databaseName)
	return nil
}

// Get mocks database document retrieval.
func (d *mockDatabaseImpl) Get(docID string, query *cloudant.DocQuery, target interface{}) error {
	content := d.client.databases[d.databaseName]
	return json.NewDecoder(bytes.NewReader(content.Docs[docID])).Decode(target)
}

// Insert mocks document insert.
func (d *mockDatabaseImpl) Insert(doc interface{}) (*cloudant.DocumentMeta, error) {
	bytes, err := cloudant.UnescapedHTMLJSONMarshal(doc)
	if err != nil {
		return nil, err
	}
	now := time.Now().Format("20060102150405") //  timestamp is used as _rev and for ID if missing

	id, err := jsonparser.GetString(bytes, "_id")
	// Create "random" _id if missing
	if err != nil {
		// _id was not supplied, creating a new one:
		id = now
		bytes, _ = jsonparser.Set(bytes, []byte("\""+id+"\""), "_id")
	}

	// Create "random" _rev
	bytes, _ = jsonparser.Set(bytes, []byte("\""+now+"\""), "_rev")

	d.client.databases[d.databaseName].Docs[id] = bytes
	meta := &cloudant.DocumentMeta{ID: id, Rev: now}

	return meta, nil
}

// InsertEscaped mocks document escaped insert.
func (d *mockDatabaseImpl) InsertEscaped(doc interface{}) (*cloudant.DocumentMeta, error) {
	bytes, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	now := time.Now().Format("20060102150405") // timestamp is used as _rev and for ID if missing

	id, err := jsonparser.GetString(bytes, "_id")
	// Create "random" _id if missing
	if err != nil {
		// _id was not supplied, creating a new one:
		id = now
		bytes, _ = jsonparser.Set(bytes, []byte("\""+id+"\""), "_id")
	}

	// Create "random" _rev
	bytes, _ = jsonparser.Set(bytes, []byte("\""+now+"\""), "_rev")

	d.client.databases[d.databaseName].Docs[id] = bytes
	meta := &cloudant.DocumentMeta{ID: id, Rev: now}

	return meta, nil
}

// InsertRaw mocks document raw insert.
func (d *mockDatabaseImpl) InsertRaw(bytes []byte) (*cloudant.DocumentMeta, error) {
	now := time.Now().Format("20060102150405") //  timestamp is used as _rev and for ID if missing

	id, err := jsonparser.GetString(bytes, "_id")
	// Create "random" _id if missing
	if err != nil {
		// _id was not supplied, creating a new one:
		id = now
		bytes, _ = jsonparser.Set(bytes, []byte("\""+id+"\""), "_id")
	}

	// Create "random" _rev
	bytes, _ = jsonparser.Set(bytes, []byte("\""+now+"\""), "_rev")
	d.client.databases[d.databaseName].Docs[id] = bytes
	meta := &cloudant.DocumentMeta{ID: id, Rev: now}

	return meta, nil
}

// Destroy hard deletes document.
func (d *mockDatabaseImpl) Destroy(docID, rev string) error {
	delete(d.client.databases[d.databaseName].Docs, docID)
	return nil
}

// List returns a channel of all documents view rows.
func (d *mockDatabaseImpl) List(q *cloudant.ViewQuery) (<-chan []byte, error) {
	results := make(chan []byte, 1000)
	for id, doc := range d.client.databases[d.databaseName].Docs {
		results <- []byte(`{"id":"` + id + `","key":"~mock~","value":"~mock~","doc":` + string(doc) + `}`)
	}

	return results, nil
}

// View returns a channel of view documents rows.
func (d *mockDatabaseImpl) View(designName, viewName string, q *cloudant.ViewQuery) (<-chan []byte, error) {
	urlStr, _ := cloudant.Endpoint(url.URL{}, "/_design/"+designName+"/_view/"+viewName, q.URLValues)

	// if mock view was supplied, get the list of IDs to include
	view, viewExists := d.client.databases[d.databaseName].Views[urlStr]

	results := make(chan []byte, 1000)
	for id, doc := range d.client.databases[d.databaseName].Docs {
		// if view does not exist, build a view out of all documents, otherwise include IDs specified in the mock view only
		if !viewExists || Contains(view, id) {
			results <- []byte(`{"id":"` + id + `","key":"~mock~","value":"~mock~","doc":` + string(doc) + `}`)
		}
	}

	return results, nil
}

// ViewRaw returns mock raw view response.
func (d *mockDatabaseImpl) ViewRaw(designName, viewName string, q *cloudant.ViewQuery) ([]byte, error) {
	urlStr, _ := cloudant.Endpoint(url.URL{}, "/_design/"+designName+"/_view/"+viewName, q.URLValues)

	// if mock view was supplied, get the list of IDs to include
	view, viewExists := d.client.databases[d.databaseName].Views[urlStr]

	docs := []string{}
	for id, doc := range d.client.databases[d.databaseName].Docs {
		// if view does not exist, build a view out of all documents, otherwise include IDs specified in the mock view only
		if !viewExists || Contains(view, id) {
			docs = append(docs, `{"id":"`+id+`","key":"~mock~","value":"~mock~","doc":`+string(doc)+`}`)
		}
	}
	return []byte(`{"rows":[` + strings.Join(docs[:], ",") + `]}`), nil
}

// Contains finds if item exists in an array.
func Contains(s interface{}, elem interface{}) bool {
	arrV := reflect.ValueOf(s)

	if arrV.Kind() == reflect.Slice {
		for i := 0; i < arrV.Len(); i++ {
			if arrV.Index(i).Interface() == elem {
				return true
			}
		}
	}
	return false
}
