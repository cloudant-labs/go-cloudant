/**
 * main -tests
 */

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"net/http/httptest"
	"testing"

	"github.com/barshociaj/go-cloudant"
	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
)

// MockResponse defines mock input and output of Cloudant calls
type MockResponse struct {
	Operation string // Expected request operation, e.g. POST
	Path      string // Expected request path, e.g. /doc/doc1
	Body      []byte // Expected request body, e.g. {"_id":"doc1"}
	Code      int    // Mock response code to send, e.g. 201 or 404
	Response  []byte // Mock response body, e.g. {"error":"not_found","reason":"missing"}
}

// CreateMockCloudantClient creates httptest server that mocks Cloudant instance for quick testing of pre-defined mock responses
func NewMockClient(mockResponses ...MockResponse) (*cloudant.Client, error) {
	// Mock
	s := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := ioutil.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			found := false
			for _, m := range mockResponses {
				if m.Operation == r.Method && m.Path == r.URL.Path && (m.Body == nil || strings.TrimSpace(string(body)) == strings.TrimSpace(string(m.Body))) {
					w.WriteHeader(m.Code)
					w.Write(m.Response)
					found = true
					if len(m.Response) > 0 {
						fmt.Println("Sending back mock response body:", string(m.Response))
					}
				}
			}
			if !found && r.URL.Path != "/_session" {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(CLOUDANT404ERROR))
				fmt.Println("Received request that has no mock responses configured, sending: " + CLOUDANT404ERROR)
			}
		}),
	)
	//defer s.Close()

	return cloudant.NewClient("N/A", "N/A", s.URL, cloudant.ClientConcurrency(50))
}

// Test SetUpServer
func TestSetUpServer(t *testing.T) {
	// Mock
	api := &API{}
	api.SetUpServer()

	// Test
	req, err := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	api.Server.Router.ServeHTTP(w, req)

	// Evaluate
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	body, _ := ioutil.ReadAll(w.Result().Body)
	data, _ := jsonparser.GetString(body, "status")
	assert.Equal(t, "ok", data)
}
