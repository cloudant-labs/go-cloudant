/**
 * controller - tests
 * -
 */

package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/barshociaj/go-cloudant/interface"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

const validDocID = "validDocID"
const invalidDocID = "invalidDocID"

var validDocJSON = []byte(`{"_id":"` + validDocID + `","_rev":"34-23412341324","foo":"bar"}`)
var postDocJSON = []byte(`{"foo":"tab"}`)

// Test GET
func TestGetAndSet(t *testing.T) {
	// Set up GIN test
	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	c, r := gin.CreateTestContext(w)
	r.Use(ErrorMiddleware)

	// Define route to test and it's dependencies
	api := &API{}

	// Initialize database connection
	// a. ... using real cloudant
	//api.Cloudant, _ = cloudanti.CreateClient(Getenv("COUCH_USER"), Getenv("COUCH_PASS"), Getenv("COUCH_HOST_URL"), 50)
	// b. ... using mock cloudant
	content := cloudanti.CloudantContent{
		Databases: map[string]cloudanti.DatabaseContent{
			DBNAME: {
				Docs: map[string][]byte{
					validDocID: validDocJSON,
				},
			},
		},
	}

	api.Cloudant, _ = cloudanti.NewMockClient(content)

	// Route to test
	r.GET("/doc/:id", api.Get)

	// Run test request
	c.Request, _ = http.NewRequest(http.MethodGet, "/doc/"+validDocID, nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code)

	// Route to test
	r.PUT("/doc/:id", api.Put)

	// Run test request
	c.Request, _ = http.NewRequest(http.MethodPut, "/doc/"+invalidDocID, bytes.NewReader(postDocJSON))
	r.ServeHTTP(w, c.Request)

	//fmt.Println("----", w)

	// Evaluate
	// Assert correct response code
	assert.Equal(t, http.StatusOK, w.Code)
}
