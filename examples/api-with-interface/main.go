/**
 * main
 */

package main

import (
	"github.com/barshociaj/go-cloudant"
	"github.com/barshociaj/go-cloudant/interface"
)

// API is global API object for injecting dependencies
type API struct {
	Server   *Server
	Cloudant cloudanti.Client
}

// DBNAME is database constant
const DBNAME = "mydb"

// NOTFOUND is error constant
const NOTFOUND = "Not found"

// INTERNALSERVERERROR is error constant
const INTERNALSERVERERROR = "Internal server error"

// BADREQUEST is error constant
const BADREQUEST = "Bad request"

// CLOUDANT404ERROR is error on document not found in DB
const CLOUDANT404ERROR = "404: {not_found, missing}"

// SetUpServer implements API Server using configured dependencies and is testable
func (api *API) SetUpServer() {
	// Initialize database connection
	api.Cloudant, _ = cloudanti.NewClient(Getenv("COUCH_USER"), Getenv("COUCH_PASS"), Getenv("COUCH_HOST_URL"), cloudant.ClientConcurrency(50))

	// Create the server
	api.Server = NewServer()

	// Add routes from routes.go
	api.Routes()
}

// main defines dependencies and is not testable
func main() {
	// Default dependencies holder
	api := &API{}

	// Set up the server
	api.SetUpServer()

	// Run server. Not used in tests
	api.Server.Run()
}
