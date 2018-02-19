package cloudant

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

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

	_, err := CreateClient(username, password, host, 5)

	if err == nil {
		t.Errorf("missing error from invalid login attempt")
	}
	if err.Error() != "failed to create session, status 401" {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestAllDBs(t *testing.T) {
	dbNames := []string{}
	client, err := makeClient()
	if err != nil {
		t.Fatalf("%s", err)
	}

	for i := 0; i < 50; i++ {
		testdbname, err := dbName()
		if err != nil {
			t.Fatalf("%s", err)
		}

		_, err = client.GetOrCreate(testdbname)
		if err != nil {
			t.Fatalf("%s", err)
		}
		dbNames = append(dbNames, testdbname)
	}
	defer func() {
		for _, name := range dbNames {
			client.Delete(name)
		}
	}()

	allDBs := map[string]bool{}
	dbList, err := client.AllDBs(&allDBsQuery{})
	for _, item := range *dbList {
		allDBs[item] = true
	}

	for _, name := range dbNames {
		if _, ok := allDBs[name]; !ok {
			t.Errorf("database not found: %s", name)
		}
	}
}

func TestAllDBs_StartKeyEndKey(t *testing.T) {
	if travis() {
		fmt.Printf("[SKIP] TestAllDBs_StartKeyEndKey requires CouchDB 2.X")
		return
	}
	dbNames := map[string]bool{}
	client, err := makeClient()
	if err != nil {
		t.Fatalf("%s", err)
	}

	dbCount := 0
	for i := 0; i < 50; i++ {
		var dbname string
		if i < 25 {
			dbname = fmt.Sprintf("aaaa%d", i)
		} else {
			dbname = fmt.Sprintf("bbbb%d", i)
			dbCount++
		}
		_, err = client.GetOrCreate(dbname)
		if err != nil {
			t.Fatalf("%s", err)
		}
		dbNames[dbname] = true
	}
	defer func() {
		for name := range dbNames {
			client.Delete(name)
		}
	}()

	q := NewAllDBsQuery().StartKey("bbbb").EndKey("bbbb5").Build()
	dbList, err := client.AllDBs(q)
	if len(*dbList) != dbCount {
		t.Errorf("expected %d databases, found %d", dbCount, len(*dbList))
	}
	for _, item := range *dbList {
		if !strings.HasPrefix(item, "bbbb") {
			t.Errorf("unexpected database name: %s", item)
		}
	}
}

func TestAllDBs_Limit(t *testing.T) {
	if travis() {
		fmt.Printf("[SKIP] TestAllDBs_Limit requires CouchDB 2.X")
	}
	dbNames := map[string]bool{}
	client, err := makeClient()
	if err != nil {
		t.Fatalf("%s", err)
	}

	dbCount := 0
	for i := 0; i < 20; i++ {
		dbname := fmt.Sprintf("aaaa%d", i)
		_, err = client.GetOrCreate(dbname)
		if err != nil {
			t.Fatalf("%s", err)
		}
		dbNames[dbname] = true
	}
	defer func() {
		for name := range dbNames {
			client.Delete(name)
		}
	}()

	query := NewAllDBsQuery().Limit(10).Build()
	values, _ := query.GetQuery()
	queryString := values.Encode()
	fmt.Print(queryString)

	dbList, err := client.AllDBs(query)
	if len(*dbList) != 10 {
		t.Errorf("expected %d databases, found %d", dbCount, len(*dbList))
	}
}
