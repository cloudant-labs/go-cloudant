package cloudant

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
)

type cloudantDocument struct {
	ID  string `json:"_id"`
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}

func dbName() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	uuid[8] = uuid[8]&^0xc0 | 0x80
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("golang-%x%x%x%x%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func makeClient() (client *CouchClient) {
	username := os.Getenv("COUCH_USER")
	password := os.Getenv("COUCH_PASS")
	client, _ = CreateClient(username, password, "https://"+username+".cloudant.com", 1)

	return client
}

func makeDatabase() (database *Database) {
	client := makeClient()
	testdbname, err := dbName()
	database, err = client.GetOrCreate(testdbname)

	if err != nil {
		fmt.Printf("Created database %s", testdbname)
	}

	return database
}

func makeDocuments(database *Database, docCount int) {
	uploader := database.Bulk(docCount, 0)
	jobs := make([]*BulkJob, docCount)
	for i := 0; i < docCount; i++ {
		foo, _ := dbName()
		jobs[i] = uploader.Upload(cloudantDocument{
			ID:  fmt.Sprintf("doc-%.3d", i+1),
			Foo: foo,
			Bar: 123,
		})
	}
	for _, job := range jobs {
		job.Wait()
	}
}
