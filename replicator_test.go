package cloudant

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRevsDiff(t *testing.T) {
	database, err := makeDatabase()
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	if err != nil {
		t.Errorf("failed to create database")
	}

	doc := &struct {
		ID  string `json:"_id"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		"mydata",
		57,
	}

	rev1, err1 := database.Set(doc)
	if err1 != nil {
		t.Error("failed to create document")
		return
	}
	if !strings.HasPrefix(rev1, "1-") {
		t.Error("got unexpected revision on create")
		return
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	doc2 := &struct {
		ID  string `json:"_id"`
		Rev string `json:"_rev"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		rev1,
		"mydata",
		57,
	}

	rev2, err2 := database.Set(doc2)
	if err2 != nil {
		t.Error("failed to update document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	fakeRev := "3-b6b61a4f380712142ea80c90f172cc1e"
	rd := RevsDiffRequestBody{}
	rd["doc-new"] = []string{rev1, rev2, fakeRev}

	missing, err := database.RevsDiff(&rd)
	if err != nil {
		t.Errorf("RevsDiff failed %s", err)
	}

	for _, revs := range *missing {
		for _, rev := range revs.Missing {
			if rev != fakeRev {
				t.Errorf("RevsDiff expcted %s but found %s", fakeRev, rev)
			}
		}
	}
}

func TestBulkGet(t *testing.T) {
	database, err := makeDatabase()
	defer func() {
		fmt.Printf("Deleting database %s", database.Name)
		database.client.Delete(database.Name)
	}()

	if err != nil {
		t.Errorf("failed to create database")
	}

	doc := &struct {
		ID  string `json:"_id"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		"mydata",
		57,
	}

	rev1, err1 := database.Set(doc)
	if err1 != nil {
		t.Error("failed to create document")
		return
	}
	if !strings.HasPrefix(rev1, "1-") {
		t.Error("got unexpected revision on create")
		return
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	doc2 := &struct {
		ID  string `json:"_id"`
		Rev string `json:"_rev"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		rev1,
		"mydata",
		57,
	}

	rev2, err2 := database.Set(doc2)
	if err2 != nil {
		t.Error("failed to update document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	bg := &BulkGetRequest{}
	bg.Add("doc-new", rev1)
	bg.Add("doc-new", rev2)

	resp, err := database.BulkGet(bg)

	count := 0
	for _, item := range resp.Results {
		count += len(item.Docs)
	}

	if count != 2 {
		t.Error("got unexpected number of revs back from BulkGet")
		return
	}
}

func TestReplication(t *testing.T) {
	database1, err := makeDatabase()
	if err != nil {
		t.Errorf("failed to create database 1")
	}
	defer func() {
		fmt.Printf("Deleting database %s", database1.Name)
		database1.client.Delete(database1.Name)
	}()

	database2, err := makeDatabase()
	if err != nil {
		t.Errorf("failed to create database 2")
	}

	defer func() {
		fmt.Printf("Deleting database %s", database2.Name)
		database2.client.Delete(database2.Name)
	}()

	doc := &struct {
		ID  string `json:"_id"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		"mydata",
		57,
	}

	rev1, err1 := database1.Set(doc)
	if err1 != nil {
		t.Error("failed to create document")
		return
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	doc2 := &struct {
		ID  string `json:"_id"`
		Rev string `json:"_rev"`
		Foo string `json:"foo"`
		Bar int    `json:"bar"`
	}{
		"doc-new",
		rev1,
		"mydata",
		57,
	}

	rev2, err2 := database.Set(doc2)
	if err2 != nil {
		t.Error("failed to update document")
	}

	// Note: lame attempt to close inconsistency window
	time.Sleep(500 * time.Millisecond)

	database1.ReplicateTo(database2, 10, 2)

	if count != 2 {
		t.Error("got unexpected number of revs back from BulkGet")
		return
	}
}
