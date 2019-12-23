package cloudant

import (
	"fmt"
	"strings"
	"testing"
)

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

		_, err = client.UseOrCreate(testdbname)
		if err != nil {
			t.Fatalf("%s", err)
		}
		dbNames = append(dbNames, testdbname)
	}
	defer func() {
		for _, name := range dbNames {
			client.Destroy(name)
		}
	}()

	allDBs := map[string]bool{}
	dbList, err := client.List(NoParams())
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
		_, err = client.UseOrCreate(dbname)
		if err != nil {
			t.Fatalf("%s", err)
		}
		dbNames[dbname] = true
	}
	defer func() {
		for name := range dbNames {
			client.Destroy(name)
		}
	}()

	params := NewDBsQuery().StartKey("bbbb").EndKey("bbbb5").Values
	dbList, err := client.List(params)
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
		return
	}
	limit := 20
	dbNames := map[string]bool{}
	client, err := makeClient()
	if err != nil {
		t.Fatalf("%s", err)
	}

	for i := 0; i < 20; i++ {
		dbname := fmt.Sprintf("aaaa%d", i)
		_, err = client.UseOrCreate(dbname)
		if err != nil {
			t.Fatalf("%s", err)
		}
		dbNames[dbname] = true
	}
	defer func() {
		for name := range dbNames {
			client.Destroy(name)
		}
	}()

	params := NewDBsQuery().Limit(limit).Values
	dbList, err := client.List(params)
	if len(*dbList) != limit {
		t.Errorf("expected %d databases, found %d", limit, len(*dbList))
	}
}
