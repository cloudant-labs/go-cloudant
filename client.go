package cloudant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"time"
)

// LogFunc is a function that logs the provided message with optional fmt.Sprintf-style arguments.
// By default, logs to the default log.Logger.
var LogFunc = log.Printf

// HTTP client timeouts
var transportTimeout = 30 * time.Second
var transportKeepAlive = 30 * time.Second
var handshakeTimeoutTLS = 10 * time.Second
var responseHeaderTimeout = 10 * time.Second
var expectContinueTimeout = 1 * time.Second

// CouchClient is the representation of a client connection
type CouchClient struct {
	username      string
	password      string
	rootURL       *url.URL
	httpClient    *http.Client
	jobQueue      chan *Job
	retryCountMax int
	retryDelayMin int
	retryDelayMax int
	workers       []*worker
	workerChan    chan chan *Job
	workerCount   int
}

// QueryBuilder is used by functions implementing Cloudant API calls
// that have many optional parameters
type QueryBuilder interface {
	GetQuery() (url.Values, error)
}

// Endpoint is a convenience function to build url-strings
func Endpoint(base url.URL, pathStr string, params url.Values) (string, error) {
	base.Path = path.Join(base.Path, pathStr)
	base.RawQuery = params.Encode()
	return base.String(), nil
}

// CreateClient returns a new client (with max. retry 3 using a random 5-30 secs delay).
func CreateClient(username, password, rootStrURL string, concurrency int) (*CouchClient, error) {
	if concurrency <= 0 {
		return nil, fmt.Errorf("Concurrency must be >= 1")
	}
	return CreateClientWithRetry(username, password, rootStrURL, concurrency, 3, 5, 30)
}

// CreateClientWithRetry returns a new client with configurable retry parameters
func CreateClientWithRetry(username, password, rootStrURL string, concurrency, retryCountMax,
	retryDelayMin, retryDelayMax int) (*CouchClient, error) {

	rand.Seed(time.Now().Unix()) // seed value for job retry start delays

	cookieJar, _ := cookiejar.New(nil)

	c := &http.Client{
		Jar: cookieJar,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   transportTimeout,
				KeepAlive: transportKeepAlive,
			}).Dial,
			TLSHandshakeTimeout:   handshakeTimeoutTLS,
			ResponseHeaderTimeout: responseHeaderTimeout,
			ExpectContinueTimeout: expectContinueTimeout,
		},
	}

	apiURL, err := url.ParseRequestURI(rootStrURL)
	if err != nil {
		return nil, err
	}

	couchClient := CouchClient{
		username:      username,
		password:      password,
		rootURL:       apiURL,
		httpClient:    c,
		jobQueue:      make(chan *Job, 100),
		retryCountMax: retryCountMax,
		retryDelayMin: retryDelayMin,
		retryDelayMax: retryDelayMax,
		workerCount:   concurrency,
	}

	startDispatcher(&couchClient) // start workers

	err = couchClient.LogIn() // create initial session
	if err != nil {
		return nil, err
	}

	return &couchClient, nil
}

// Delete deletes a specified database.
func (c *CouchClient) Delete(databaseName string) error {
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

// AllDBs returns a list of all DBs
func (c *CouchClient) AllDBs(args *allDBsQuery) (*[]string, error) {
	params, err := args.GetQuery()
	if err != nil {
		return nil, err
	}

	urlStr, err := Endpoint(*c.rootURL, "/_all_dbs", params)
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

// Get returns a database. It is assumed to exist.
func (c *CouchClient) Get(databaseName string) (*Database, error) {
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

// GetOrCreate returns a database.
// If the database doesn't exist on the server then it will be created.
func (c *CouchClient) GetOrCreate(databaseName string) (*Database, error) {
	database, err := c.Get(databaseName)
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

// LogIn creates a session.
func (c *CouchClient) LogIn() error {
	sessionURL := c.rootURL.String() + "/_session"

	data := url.Values{}
	data.Add("name", c.username)
	data.Add("password", c.password)

	req, err := http.NewRequest("POST", sessionURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	job := CreateJob(req)
	defer job.Close()

	job.isLogin = true // don't retry login on 401

	c.Execute(job)
	job.Wait() // wait for job to complete

	if job.error != nil {
		return job.error
	}

	if job.response.StatusCode != 200 {
		return fmt.Errorf("failed to create session, status %d", job.response.StatusCode)
	}

	return nil // success
}

// LogOut deletes the current session.
func (c *CouchClient) LogOut() {
	sessionURL := c.rootURL.String() + "/_session"
	job, _ := c.request("DELETE", sessionURL, nil) // ignore failures
	job.Close()
}

func (c *CouchClient) request(method, path string, body io.Reader) (job *Job, err error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	if req.Method == "POST" {
		req.Header.Add("Content-Type", "application/json") // add Content-Type for POSTs
	}

	job = CreateJob(req)

	c.Execute(job)
	job.Wait()

	if job.error != nil {
		return job, job.error
	}

	return job, nil
}

// Execute submits a job for execution.
// The client must call `job.Wait()` before attempting access the response attribute.
// Always call `job.Close()` to ensure the underlying connection is terminated.
func (c *CouchClient) Execute(job *Job) { c.jobQueue <- job }

// Ping can be used to check whether a server is alive.
// It sends an HTTP HEAD request to the server's URL.
func (c *CouchClient) Ping() (err error) {
	job, err := c.request("HEAD", c.rootURL.String(), nil)
	job.Close()

	return
}

// Stop kills all running workers.
// Once called the client is no longer able to execute new jobs.
func (c *CouchClient) Stop() {
	for _, worker := range c.workers {
		worker.stop()
	}

}
