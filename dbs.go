package cloudant

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
)

// Database holds a reference to an authenticated client connection and the
// name of a remote database
type Database struct {
	client *CouchClient
	Name   string
	URL    *url.URL
}

// Info represents the account meta-data
type Info struct {
	IsCompactRunning  bool   `json:"compact_running"`
	DBName            string `json:"db_name"`
	DiskFromatVersion int    `json:"disk_format_version"`
	DiskSize          int    `json:"disk_size"`
	DocCount          int    `json:"doc_count"`
	DocDelCount       int    `json:"doc_del_count"`
	PurgeSeq          int    `json:"purge_seq"`
	Sizes             Sizes  `json:"sizes"`
	UpdateSeq         string `json:"update_seq"`
}

// Sizes represents the sizes part of database info
type Sizes struct {
	File     int `json:"file"`
	External int `json:"external"`
	Active   int `json:"active"`
}

// List returns a list of all DBs
func (c *CouchClient) List(q *DBsQuery) (*[]string, error) {
	urlStr, err := Endpoint(*c.rootURL, "/_all_dbs", q.URLValues)
	if err != nil {
		return nil, err
	}

	job, err := c.request("GET", urlStr, nil)
	defer job.Close()
	if err != nil {
		return nil, err
	}

	err = expectedReturnCodes(job, 200)
	if err != nil {
		return nil, err
	}

	vals := &[]string{}
	err = json.NewDecoder(job.response.Body).Decode(vals)
	return vals, err
}

// Info returns database information.
// See https://console.bluemix.net/docs/services/Cloudant/api/database.html#getting-database-details
func (c *CouchClient) Info(databaseName string) (*Info, error) {
	d, err := c.Use(databaseName)
	if err != nil {
		return nil, err
	}

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

// Destroy deletes a specified database.
func (c *CouchClient) Destroy(databaseName string) error {
	databaseURL, err := url.Parse(c.rootURL.String())
	if err != nil {
		return err
	}

	databaseURL.Path = path.Join(databaseURL.Path, databaseName)

	job, err := c.request("DELETE", databaseURL.String(), nil)
	defer job.Close()

	if err != nil {
		return fmt.Errorf("failed to delete database %s, %s", databaseName, err)
	}

	if job.response.StatusCode != 200 {
		return fmt.Errorf(
			"failed to delete database %s, status %d", databaseName, job.response.StatusCode)
	}

	return nil
}

// Exists checks the existence of a specified database.
// Returns true if the database exists, else false.
func (c *CouchClient) Exists(databaseName string) (bool, error) {
	databaseURL, err := url.Parse(c.rootURL.String())
	if err != nil {
		return false, err
	}

	job, err := c.request("HEAD", databaseURL.String(), nil)
	defer job.Close()

	if err != nil {
		return false, fmt.Errorf("failed to query server: %s", err)
	}

	return job.response.StatusCode == 200, nil
}

// Use returns a database. It is assumed to exist.
func (c *CouchClient) Use(databaseName string) (*Database, error) {
	databaseURL, err := url.Parse(c.rootURL.String())
	if err != nil {
		return nil, err
	}

	databaseURL.Path += "/" + databaseName

	database := &Database{
		client: c,
		Name:   databaseName,
		URL:    databaseURL,
	}

	return database, nil
}

// UseOrCreate returns a database.
// If the database doesn't exist on the server then it will be created.
func (c *CouchClient) UseOrCreate(databaseName string) (*Database, error) {
	database, err := c.Use(databaseName)
	if err != nil {
		return nil, err
	}

	job, err := c.request("PUT", database.URL.String(), nil)
	defer job.Close()

	if err != nil {
		return nil, fmt.Errorf("failed to create database: %s", err)
	}

	if job.error != nil {
		return nil, fmt.Errorf("failed to create database: %s", job.error)
	}

	if job.response.StatusCode != 201 && job.response.StatusCode != 412 {
		return nil, fmt.Errorf(
			"failed to create database, status %d", job.response.StatusCode)
	}

	return database, nil
}
