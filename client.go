package cloudant

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

// LogFunc is a function that logs the provided message with optional fmt.Sprintf-style arguments.
// By default, logs to the default log.Logger.
var LogFunc func(string, ...interface{}) = log.Printf

// HTTP client timeouts
var TransportTimeout time.Duration = 30 * time.Second
var TransportKeepAlive time.Duration = 30 * time.Second
var TLSHandshakeTimeout time.Duration = 10 * time.Second
var ResponseHeaderTimeout time.Duration = 10 * time.Second
var ExpectContinueTimeout time.Duration = 1 * time.Second

type CouchClient struct {
	username   string
	password   string
	apiURL     *url.URL
	httpClient *http.Client
	jobQueue   chan *Job
	workers    []*worker
	workerChan chan chan *Job
}

func CreateClient(username, password, rootURL string, concurrency int) (*CouchClient, error) {
	cookieJar, _ := cookiejar.New(nil)

	c := &http.Client{
		Jar: cookieJar,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   TransportTimeout,
				KeepAlive: TransportKeepAlive,
			}).Dial,
			TLSHandshakeTimeout:   TLSHandshakeTimeout,
			ResponseHeaderTimeout: ResponseHeaderTimeout,
			ExpectContinueTimeout: ExpectContinueTimeout,
		},
	}

	apiURL, err := url.ParseRequestURI(rootURL)
	if err != nil {
		return nil, err
	}

	couchClient := CouchClient{
		username:   username,
		password:   password,
		apiURL:     apiURL,
		httpClient: c,
		jobQueue:   make(chan *Job, 100),
	}

	startDispatcher(&couchClient, concurrency) // start workers

	couchClient.LogIn() // create initial session

	return &couchClient, nil
}

func (c *CouchClient) GetOrCreateDatabase(databaseName string) (*Database, error) {
	databaseURL, err := url.Parse(c.apiURL.String())
	if err != nil {
		return nil, err
	}

	databaseURL.Path += "/" + databaseName

	job, err := c.request("PUT", databaseURL.String(), nil)
	defer job.Close()

	if err != nil {
		return nil, fmt.Errorf("unable to create database: %s", err)
	}

	if job.error != nil {
		return nil, fmt.Errorf("unable to create database: %s", job.error)
	}

	if job.response.StatusCode == 201 || job.response.StatusCode == 412 {
		database := &Database{
			client:       c,
			DatabaseName: databaseName,
			databaseURL:  databaseURL,
		}

		return database, nil
	} else {
		return nil, errors.New("unable to create database")
	}
}

func (c *CouchClient) LogIn() error {
	sessionURL := c.apiURL.String() + "/_session"

	data := url.Values{}
	data.Set("name", c.username)
	data.Set("password", c.password)

	req, err := http.NewRequest("POST", sessionURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	job := CreateJob(req)
	defer job.Close()

	c.Execute(job)
	job.Wait() // wait for job to complete

	if job.error != nil {
		return job.error
	}

	return nil // success
}

func (c *CouchClient) LogOut() {
	sessionURL := c.apiURL.String() + "/_session"
	job, _ := c.request("DELETE", sessionURL, nil) // ignore failures
	defer job.Close()
}

func (c *CouchClient) request(method, path string, body io.Reader) (job *Job, err error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return
	}

	job = CreateJob(req)

	c.Execute(job)
	job.Wait()

	return
}

func (c *CouchClient) Execute(job *Job) { c.jobQueue <- job }

func (c *CouchClient) Ping() (err error) {
	job, err := c.request("HEAD", c.apiURL.String(), nil)
	defer job.Close()

	return
}

func (c *CouchClient) Stop() {
	for _, worker := range c.workers {
		worker.stop()
	}

}
