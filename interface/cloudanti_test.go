package cloudanti

import (
	"os"
	"testing"

	"github.com/barshociaj/go-cloudant"
)

func TestCloudantInterface(t *testing.T) {
	username := os.Getenv("COUCH_USER")
	password := os.Getenv("COUCH_PASS")
	host := os.Getenv("COUCH_HOST_URL")
	dbName := "testdb"

	client, _ := NewClient(username, password, host)

	// Test creating a database
	db, _ := client.UseOrCreate(dbName)

	// Define document struct and example test doc
	type Doc struct {
		ID  string `json:"_id,omitempty"`
		Rev string `json:"_rev,omitempty"`
		Foo string `json:"foo"`
	}
	doc := Doc{
		Foo: "testValue",
	}

	// Test creating a doc
	meta, err := db.Insert(doc)
	if err != nil {
		t.Fatal("could not insert doc")
	}
	getDoc := new(Doc)

	// Test retrieving the created doc
	err = db.Get(meta.ID, cloudant.NewDocQuery(), &getDoc)
	if err != nil {
		t.Fatal("could not get doc")
	}

	// Test deleting the created doc
	err = db.Destroy(meta.ID, meta.Rev)
	if err != nil {
		t.Fatal("could not delete doc")
	}

	// Test deleting the created database
	err = client.Destroy(dbName)
	if err != nil {
		t.Fatal("could not delete database")
	}
}
