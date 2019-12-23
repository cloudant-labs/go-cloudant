package cloudant

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestDatabase_StaticChanges(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	makeDocuments(database, 1000)

	changes, err := database.Changes(NoParams())
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-changes
		if more {
			i++
		} else {
			break
		}
	}

	if 1000 != i {
		t.Errorf("unexpected number of changes received %d", i)
	}
}

func TestDatabase_ChangesIncludeDocs(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	makeDocuments(database, 1000)
	params := NewChangesQuery().
		IncludeDocs().
		Values

	changes, err := database.Changes(params)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		ch, more := <-changes
		if more {
			i++
		} else {
			break
		}
		if ch.Doc == nil {
			t.Error("Missing doc body")
		}
	}

	if 1000 != i {
		t.Errorf("unexpected number of changes received %d", i)
	}
}

func TestDatabase_ContinousChanges(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	makeDocuments(database, 1000)

	params := NewChangesQuery().
		Feed("continuous").
		Timeout(10).
		Values

	changes, err := database.Changes(params)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-changes
		if more {
			i++
		} else {
			break
		}
	}

	if 1000 != i {
		t.Errorf("unexpected number of changes received %d", i)
	}
}

func TestDatabase_ChangesSeqInterval(t *testing.T) {
	database, err := makeDatabase()
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Destroy(database.Name)
	}()

	makeDocuments(database, 1000)

	params := NewChangesQuery().
		SeqInterval(100).
		Values

	changes, err := database.Changes(params)
	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		_, more := <-changes
		if more {
			i++
		} else {
			break
		}
	}

	if 1000 != i {
		t.Errorf("unexpected number of changes received %d", i)
	}
}

// TestDatabase_ChangesCouchDB16 checks that we can read old-style changes feeds
// that uses a sequence ID which is an integer
func TestDatabase_ChangesCouchDB16(t *testing.T) {
	data1 := []byte(`{"seq":59,"id":"5100a7174427c7dfc1ecc5971949f201","changes":[{"rev":"1-cd6870b027e3a728bce927d4a1e0b3ab"}]}`)
	data2 := []byte(`{"seq":"59","id":"5100a7174427c7dfc1ecc5971949f201","changes":[{"rev":"1-cd6870b027e3a728bce927d4a1e0b3ab"}]}`)

	cr1 := &ChangeRow{}
	if err := json.Unmarshal(data1, cr1); err != nil {
		t.Error(err)
	}

	cr2 := &ChangeRow{}
	if err := json.Unmarshal(data2, cr2); err != nil {
		t.Error(err)
	}

	if cr1.Seq != cr2.Seq {
		t.Error("failed to parse CouchDB1.6-formatted changes data")
	}
}
