/**
 * main
 * - Example using `cloudanti` interface wrappper
 */

package main

import (
	"log"
	"os"

	"github.com/barshociaj/go-cloudant"

	"github.com/barshociaj/go-cloudant/interface"
	"github.com/joho/godotenv"
)

// The Item struct used to prepare the data for Cloudant insert
type Item struct {
	ID      string `json:"_id,omitempty"`
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}

// DB holds database name
const DB = "items"

var myItem = Item{
	ID:      "myItemID",
	Score:   1,
	Comment: "Number 1",
}

func create(client cloudanti.Client) *cloudant.DocumentMeta {
	// Try to get the database specified in the .env file, if it does not exist it, create it
	db, err4 := client.UseOrCreate(DB)
	if err4 != nil {
		log.Fatal("Error getting or creating Cloudant DB", err4)
	}

	// Create a document in the items DB
	result, err5 := db.Insert(myItem)
	if err5 != nil {
		log.Fatal("Error creating a document", err5)
	}

	return result
}

func main() {

	// Load Cloudant environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env file does not exist", err)
	}

	// Set the Cloudant variables from the environment
	cloudantURL := os.Getenv("CLOUDANT_URL")
	cloudantUser := os.Getenv("CLOUDANT_USER")
	cloudantPass := os.Getenv("CLOUDANT_PASS")

	// Create a Cloudant Client
	client, err2 := cloudanti.NewClient(cloudantUser, cloudantPass, cloudantURL)
	if err2 != nil {
		log.Fatal("Can not connect to Cloudant", err2)
	}

	meta := create(client)

	// Log the _id and revision number
	log.Println(meta)
}
