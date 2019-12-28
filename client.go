package cloudant

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"runtime/debug"
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

// Client is the representation of a client connection
type Client struct {
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
	version       string
}

// ClientOption is a functional option setter for Client
type ClientOption func(*Client)

// ClientConcurrency overrides default workerCount Client option
func ClientConcurrency(workerCount int) ClientOption {
	return func(c *Client) {
		if workerCount > 0 {
			c.workerCount = workerCount
		}
	}
}

// ClientRetryCountMax overrides default retryCountMax Client option
func ClientRetryCountMax(retryCountMax int) ClientOption {
	return func(c *Client) {
		c.retryCountMax = retryCountMax
	}
}

// ClientRetryDelayMin overrides default retryDelayMin Client option
func ClientRetryDelayMin(retryDelayMin int) ClientOption {
	return func(c *Client) {
		c.retryDelayMin = retryDelayMin
	}
}

// ClientRetryDelayMax overrides default retryDelayMax Client option
func ClientRetryDelayMax(retryDelayMax int) ClientOption {
	return func(c *Client) {
		c.retryDelayMax = retryDelayMax
	}
}

// ClientHTTPClient overrides default httpClient option
func ClientHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// Endpoint is a convenience function to build url-strings
func Endpoint(base url.URL, pathStr string, params url.Values) (string, error) {
	base.Path = path.Join(base.Path, pathStr)
	base.RawQuery = params.Encode()
	return base.String(), nil
}

// NewClient returns a new Cloudant client
func NewClient(username, password, rootStrURL string, options ...ClientOption) (*Client, error) {

	rand.Seed(time.Now().Unix()) // seed value for job retry start delays

	cookieJar, _ := cookiejar.New(nil)

	// Get current module version for pool request identification
	bi, ok := debug.ReadBuildInfo()
	var version string
	if ok {
		version = bi.Main.Version
	} else {
		version = "(devel)"
	}

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

	client := Client{
		username:      username,
		password:      password,
		rootURL:       apiURL,
		httpClient:    c,
		jobQueue:      make(chan *Job, 100),
		retryCountMax: 3,  // default triple retry
		retryDelayMin: 5,  // default minimum retry delay of 5 seconds
		retryDelayMax: 30, // default maximum retry delay of 30 seconds
		workerCount:   5,  // default concurrency of 5 workers
		version:       version,
	}

	// Apply functional options
	for _, f := range options {
		f(&client)
	}

	startDispatcher(&client) // start workers

	err = client.LogIn() // create initial session
	if err != nil {
		return nil, err
	}

	return &client, nil
}

// LogIn creates a session.
func (c *Client) LogIn() error {
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
func (c *Client) LogOut() {
	sessionURL := c.rootURL.String() + "/_session"
	job, _ := c.request("DELETE", sessionURL, nil) // ignore failures
	job.Close()
}

func (c *Client) request(method, path string, body io.Reader) (job *Job, err error) {
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
func (c *Client) Execute(job *Job) { c.jobQueue <- job }

// Ping can be used to check whether a server is alive.
// It sends an HTTP HEAD request to the server's URL.
func (c *Client) Ping() (err error) {
	job, err := c.request("HEAD", c.rootURL.String(), nil)
	job.Close()

	return
}

// Stop kills all running workers.
// Once called the client is no longer able to execute new jobs.
func (c *Client) Stop() {
	for _, worker := range c.workers {
		worker.stop()
	}

}
