package cloudant

import (
	"fmt"
	"testing"
)

func TestDatabase_List(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	makeDocuments(database, 1000)

	q := NewViewQuery().
		StartKey("doc-450").
		EndKey("doc-500")

	rows, err := database.List(q)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-rows
		if more {
			i++
		} else {
			break
		}
	}

	if 51 != i {
		t.Errorf("unexpected number of rows received %d", i)
	}
}

func TestDatabase_AllDocKeys(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	makeDocuments(database, 1000)

	keys := []string{
		"doc-097",
		"doc-034",
		"doc-997",
	}

	q := NewViewQuery().
		Keys(keys)

	rows, err := database.List(q)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-rows
		if more {
			i++
		} else {
			break
		}
	}

	if 3 != i {
		t.Errorf("unexpected number of rows received %d", i)
	}
}

func TestDatabase_AllDocKey(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	makeDocuments(database, 100)

	q := NewViewQuery().
		Key("doc-032")

	rows, err := database.List(q)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-rows
		if more {
			i++
		} else {
			break
		}
	}

	if 1 != i {
		t.Errorf("unexpected number of rows received %d", i)
	}
}

func TestView(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	makeDocuments(database, 15)

	_, err = database.InsertRaw([]byte(`
	{
		"_id": "_design/test_design_doc",
		"language": "javascript",
		"views": {
			"start_with_one": {
				"map": "function (doc) { if (doc._id.indexOf('-01') > 0) emit(doc._id, doc.foo) }"
			}
		}
	  }`))
	if err != nil {
		t.Error(err)
	}

	keys := []string{
		"doc-002", // does not match view, will not be returned
		"doc-009", // does not match view, will not be returned
		"doc-011", // matches view, will be returned
		"doc-014", // matches view, will be returned
	}

	q := NewViewQuery().
		Keys(keys)

	rows, err := database.View("test_design_doc", "start_with_one", q)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-rows
		if more {
			i++
		} else {
			break
		}
	}

	if 2 != i {
		t.Errorf("unexpected number of rows received from view - %d", i)
	}
}
