/**
 * main -tests
 */

package main

import (
	"io/ioutil"
	"net/http"

	"net/http/httptest"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
)

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
