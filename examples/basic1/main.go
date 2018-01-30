package main

import (
	"log"
	"os"

	cldt "github.com/cloudant-labs/go-cloudant"
	"github.com/joho/godotenv"
)

// The Item struct used to prepare the data for Cloudant insert
type Item struct {
	ID      string `json:"_id,omitempty"`
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}

var cloudant *cldt.CouchClient
var cloudantURL string
var cloudantDB string
var cloudantUser string
var cloudantPass string

func main() {

	// Load Cloudant environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file does not exist")
		log.Fatal(err)
	}

	// Set the Cloudant variables from the environment
	cloudantURL = os.Getenv("CLOUDANT_URL")
	cloudantDB = os.Getenv("CLOUDANT_DB")
	cloudantUser = os.Getenv("CLOUDANT_USER")
	cloudantPass = os.Getenv("CLOUDANT_PASS")

	// Create a Cloudant Client
	cloudant, err2 := cldt.CreateClient(cloudantUser, cloudantPass, cloudantURL, 5)
	if err2 != nil {
		log.Println("Can not connect to Cloudant")
		log.Fatal(err2)
	} else {
		log.Println("Connected to Cloudant Successfully")
	}

	// Verify we have an active connection to the server
	err3 := cloudant.Ping()
	if err3 != nil {
		log.Println("Cloudant Ping failed")
		log.Fatal(err3)
	}

	// Try to get the database specified in the .env file, if it does not exist it, create it
	db, err4 := cloudant.GetOrCreate(cloudantDB)
	if err4 != nil {
		log.Println("Error getting or creating Cloudant DB")
		log.Fatal(err4)
	}

	// Create a document in the items DB
	var item Item
	item.Score = 1
	item.Comment = "Number 1"

	result, err5 := db.Set(item)
	if err5 != nil {
		log.Println("Error creating a document")
		log.Fatal(err5)
	}
	// Log the _id and revision number
	log.Println(result)
}
