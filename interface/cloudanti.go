/**
 * Cloudant interface
 * - Provides a convenience interface wrapper to the main library
 * - Use directly or as an example for your own implementation
 */

package cloudanti

import (
	"github.com/cloudant-labs/go-cloudant"
)

// Client is the main interface for Cloudant instance operations
type Client interface {
	Use(string) (Database, error)
	UseOrCreate(string) (Database, error)
	Destroy(string) error
	Exists(databaseName string) (bool, error)
	Info(databaseName string) (*cloudant.Info, error)
}

// Database is an interface for Cloudant database operations
type Database interface {
	Get(string, *cloudant.DocQuery, interface{}) error
	Insert(interface{}) (*cloudant.DocumentMeta, error)
	InsertEscaped(document interface{}) (*cloudant.DocumentMeta, error)
	InsertRaw(jsonDocument []byte) (*cloudant.DocumentMeta, error)
	Destroy(string, string) error
	List(q *cloudant.ViewQuery) (<-chan []byte, error)
	View(designName, viewName string, q *cloudant.ViewQuery) (<-chan []byte, error)
	ViewRaw(string, string, *cloudant.ViewQuery) ([]byte, error)
}

// NewClient returns a new Cloudanti client.
func NewClient(username, password, rootStrURL string, options ...cloudant.ClientOption) (Client, error) {
	client, err := cloudant.NewClient(username, password, rootStrURL, options...)
	return &clientImpl{
		client: client,
	}, err
}

type clientImpl struct {
	client *cloudant.Client
}

type databaseImpl struct {
	client   *clientImpl
	database *cloudant.Database
}

// Destroy deletes a specified database.
func (c *clientImpl) Destroy(databaseName string) error {
	return c.client.Destroy(databaseName)
}

// Exists checks the existence of a specified database.
// Returns true if the database exists, else false.
func (c *clientImpl) Exists(databaseName string) (bool, error) {
	return c.client.Exists(databaseName)
}

// Info returns database information.
// See https://console.bluemix.net/docs/services/Cloudant/api/database.html#getting-database-details
func (c *clientImpl) Info(databaseName string) (*cloudant.Info, error) {
	return c.client.Info(databaseName)
}

// Use returns a database. It is assumed to exist.
func (c *clientImpl) Use(databaseName string) (Database, error) {
	db, err := c.client.Use(databaseName)
	return &databaseImpl{
		client:   c,
		database: db,
	}, err
}

// UseOrCreate returns a database.
// If the database doesn't exist on the server then it will be created.
func (c *clientImpl) UseOrCreate(databaseName string) (Database, error) {
	db, err := c.client.UseOrCreate(databaseName)
	return &databaseImpl{
		client:   c,
		database: db,
	}, err
}

// Get a document from the database.
func (d *databaseImpl) Get(documentID string, q *cloudant.DocQuery, target interface{}) error {
	return d.database.Get(documentID, q, target)
}

// Insert a document without escaped HTML.
func (d *databaseImpl) Insert(document interface{}) (*cloudant.DocumentMeta, error) {
	return d.database.Insert(document)
}

// InsertEscaped a document with escaped HTML.
func (d *databaseImpl) InsertEscaped(document interface{}) (*cloudant.DocumentMeta, error) {
	return d.database.InsertEscaped(document)
}

// InsertRaw posts raw input to Cloudant.
// Input may have json attributes '_id' and '_rev'.
// If no '_id' is given the database will generate one for you.
func (d *databaseImpl) InsertRaw(jsonDocument []byte) (*cloudant.DocumentMeta, error) {
	return d.database.InsertRaw(jsonDocument)
}

// Destroy a document with a specified revision.
func (d *databaseImpl) Destroy(documentID, rev string) error {
	return d.database.Destroy(documentID, rev)
}

// List returns a channel of all documents in which matching row types can be received.
func (d *databaseImpl) List(q *cloudant.ViewQuery) (<-chan []byte, error) {
	return d.database.List(q)
}

// View returns a channel of view documents in which matching row types can be received.
func (d *databaseImpl) View(designName, viewName string, q *cloudant.ViewQuery) (<-chan []byte, error) {
	return d.database.View(designName, viewName, q)
}

// ViewRaw returns raw view response.
func (d *databaseImpl) ViewRaw(designName, viewName string, q *cloudant.ViewQuery) ([]byte, error) {
	return d.database.ViewRaw(designName, viewName, q)
}
