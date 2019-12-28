/**
 * Cloudant interface
 * - Provides a convenience interface wrapper to the main library
 * - Use directly or as an example for your own implementation
 */

package cloudanti

import (
	"github.com/barshociaj/go-cloudant"
)

// Client .
type Client interface {
	Use(string) (Database, error)
	UseOrCreate(string) (Database, error)
	Destroy(string) error
}

// Database .
type Database interface {
	Get(string, *cloudant.DocQuery, interface{}) error
	Insert(interface{}) (*cloudant.DocumentMeta, error)
	Destroy(string, string) error
	ViewRaw(string, string, *cloudant.ViewQuery) ([]byte, error)
}

// NewClient returns a new client (with max. retry 3 using a random 5-30 secs delay).
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

// Use points to a database as in NodeJS nano library
func (c *clientImpl) Use(databaseName string) (Database, error) {
	db, err := c.client.Use(databaseName)
	return &databaseImpl{
		client:   c,
		database: db,
	}, err
}

// Use points to a database and creates it if necessary
func (c *clientImpl) UseOrCreate(databaseName string) (Database, error) {
	db, err := c.client.UseOrCreate(databaseName)
	return &databaseImpl{
		client:   c,
		database: db,
	}, err
}

// Destroy deletes a database
func (c *clientImpl) Destroy(databaseName string) error {
	return c.client.Destroy(databaseName)
}

// Get gets a document
func (d *databaseImpl) Get(documentID string, q *cloudant.DocQuery, target interface{}) error {
	return d.database.Get(documentID, q, target)
}

// Insert adds a document
func (d *databaseImpl) Insert(document interface{}) (*cloudant.DocumentMeta, error) {
	return d.database.Insert(document)
}

// Destroy deletes a document
func (d *databaseImpl) Destroy(documentID, rev string) error {
	return d.database.Destroy(documentID, rev)
}

// ViewRaw returns raw view response
func (d *databaseImpl) ViewRaw(designName, viewName string, q *cloudant.ViewQuery) ([]byte, error) {
	return d.database.ViewRaw(designName, viewName, q)
}
