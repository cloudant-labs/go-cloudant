/**
 * main
 * - Example using Cloudant library directly
 */

package main

import (
	"log"
	"os"

	"github.com/barshociaj/go-cloudant"
	"github.com/joho/godotenv"
)

// The Item struct used to prepare the data for Cloudant insert
type Item struct {
	ID      string `json:"_id,omitempty"`
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}

func main() {

	// Load Cloudant environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env file does not exist", err)
	}

	// Set the Cloudant variables from the environment
	cloudantUser := os.Getenv("COUCH_USER")
	cloudantPass := os.Getenv("COUCH_PASS")
	cloudantURL := os.Getenv("COUCH_HOST_URL")
	cloudantDB := os.Getenv("COUCH_DB")

	// Create a Cloudant Client
	client, err2 := cloudant.NewClient(cloudantUser, cloudantPass, cloudantURL)
	if err2 != nil {
		log.Fatal("Failed to connect to Cloudant", err2)
	} else {
		log.Println("Connected to Cloudant successfully")
	}

	// Verify we have an active connection to the server
	err3 := client.Ping()
	if err3 != nil {
		log.Fatal("Cloudant Ping failed", err3)
	}

	// Try to get the database specified in the .env file, if it does not exist it, create it
	db, err4 := client.UseOrCreate(cloudantDB)
	if err4 != nil {
		log.Fatal("Error getting or creating Cloudant DB", err4)
	}

	// Create a document in the items DB
	var item Item
	item.Score = 1
	item.Comment = "Number 1"

	result, err5 := db.Insert(item)
	if err5 != nil {
		log.Fatal("Error creating a document", err5)
	}

	// Log the _id and revision number
	log.Println(result)
}
