package cloudant

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

func setupDatabase() (database *Database) {
	client := setupClient()
	setupMocks([]*http.Response{mock412})

	database, _ = client.GetOrCreate(testDatabaseName)

	return
}

func TestDatabase_Changes(t *testing.T) {
	database := setupDatabase()

	mockChanges := "{\"results\":[\n" +
		"{\"seq\":\"1-xxxxx\",\"id\":\"doc1\",\"changes\":[{\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}]},\n" +
		"{\"seq\":\"2-xxxxx\",\"id\":\"doc2\",\"changes\":[{\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}]},\n" +
		"{\"seq\":\"3-xxxxx\",\"id\":\"doc3\",\"changes\":[{\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}]},\n" +
		"{\"seq\":\"4-xxxxx\",\"id\":\"doc4\",\"changes\":[{\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}]},\n" +
		"{\"seq\":\"5-xxxxx\",\"id\":\"doc5\",\"changes\":[{\"rev\":\"1-967a00dff5e02add41819138abb3284d\"}]}\n" +
		"],\n" +
		"\"last_seq\":\"5-xxxxx\",\"pending\":0}"

	mockChangesResponse := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(mockChanges))),
	}

	setupMocks([]*http.Response{mockChangesResponse})

	changes, err := database.Changes()

	if err != nil {
		t.Error(err)
	}

	i := 0
	for {
		change, more := <-changes
		if more {
			i += 1
			if fmt.Sprintf("doc%d", i) != change.Id  {
				t.Errorf("unexpected change id %s", change.Id)
			}
			if fmt.Sprintf("%d-xxxxx", i) != change.Seq {
				t.Errorf("unexpected change seq %s", change.Id)
			}
			if "1-967a00dff5e02add41819138abb3284d" != change.Rev {
				t.Errorf("unexpected rev value %s", change.Rev)
			}
		} else {
			break
		}
	}

	if 5 != i {
		t.Errorf("unexpected number of changes received %d", i)
	}

}
