package cloudant

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"testing"
)

/* UTILITIES */

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

func makeClient() (*Client, error) {
	username := os.Getenv("COUCH_USER")
	password := os.Getenv("COUCH_PASS")
	host := os.Getenv("COUCH_HOST_URL")

	if username == "" || password == "" {
		return nil, fmt.Errorf("Expected env vars COUCH_USER and COUCH_PASS to be set")
	}

	if host == "" {
		host = "https://" + username + ".cloudant.com"
	}

	return NewClient(username, password, host)
}

func makeDatabase() (*Database, error) {
	client, err := makeClient()
	if err != nil {
		return nil, err
	}
	testdbname, err := dbName()
	if err != nil {
		return nil, err
	}

	return client.UseOrCreate(testdbname)
}

func makeDocuments(database *Database, docCount int) {
	uploader := database.Bulk(docCount, -1, 0)
	for i := 0; i < docCount; i++ {
		foo, _ := dbName()
		uploader.Upload(cloudantDocument{
			ID:  fmt.Sprintf("doc-%.3d", i+1),
			Foo: foo,
			Bar: 123,
		})
	}
	uploader.Flush()
}

func travis() bool {
	return os.Getenv("TRAVIS") == "true"
}

/* TESTS */

func TestInvalidLogin(t *testing.T) {
	username := os.Getenv("COUCH_USER")
	password := "wR0ng_pa$$w0rd"
	host := os.Getenv("COUCH_HOST_URL")

	if host == "" {
		host = "https://" + username + ".cloudant.com"
	}
	if username == "" {
		t.Fatalf("expected env var COUCH_USER to be set")
	}

	_, err := NewClient(username, password, host)

	if err == nil {
		t.Errorf("missing error from invalid login attempt")
	}
	if err.Error() != "failed to create session, status 401" {
		t.Errorf("unexpected error message: %s", err)
	}
}
