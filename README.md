# go-cloudant

A [Cloudant](https://cloudant.com/) library for Golang. 

Forked from cloudant-labs and modified to align closer to NodeJS `nano` library. __This library is not complete, may change in incompatible ways in future versions, and comes with no support.__

[![Build Status](https://travis-ci.org/barshociaj/go-cloudant.svg?branch=master)](https://travis-ci.org/barshociaj/go-cloudant)

Features:
- Session authentication
- Keep-Alive & Connection Pooling
- Configurable request retrying
- Hard limit on request concurrency
- Stream  `/_changes`, `/_all_docs`, and other views
- Manage `/_bulk_docs` uploads

## Installation

```bash
go get github.com/barshociaj/go-cloudant
```

## Getting Started

```go
// create a Cloudant client with default configuration:
//   - concurrency:                     5
//   - maximum retries per request:     3
//   - random retry delay minimum:      5  seconds
//   - random retry delay maximum:      30 seconds
client, err := cloudant.NewClient("user123", "pa55w0rd01", "https://user123.cloudant.com")

// OR provide any number of custom client options
//
// client, err := cloudant.NewClient("user123", "pa55w0rd01", "https://user123.cloudant.com", cloudant.ClientConcurrency(20), cloudant.ClientRetryCountMax(5), cloudant.ClientRetryDelayMin(10), cloudant.ClientRetryDelayMax(60))
```

## Database functions

### `client.Destroy(dbName)`

Delete existing database
```go
err := client.Destroy("my_db")
```

### `client.Exists(dbName)`

Checks if database exists
```go
exists, err := client.Exists("my_db")
```

### `client.Info(dbName)`

Retrieve database info
```go
info, err := client.Info("my_db")
fmt.Println(info.DocCount) // prints the number of documents in the database

```

### `client.List(dbsQuery)`

List existing databases
```go
dbList, err := client.List(cloudant.NewDBsQuery())
for _, name := range *dbList {
    fmt.Println(name) // prints database names
}
```

### `client.Use(dbName)`

Use existing database:
```go
db, err := client.Use("my_database")
```

### `client.UseOrCreate(dbName)`

Create (if does not exist) and use a database:
```go
db, err := client.UseOrCreate("my_database")
```

### `db.Changes(changesQuery)`

Get changes feed from the database

```go
q := cloudant.NewChangesQuery().IncludeDocs()

changes, err := db.Changes(q)

for {
    change, more := <-changes
    if more {
        fmt.Println(change.Seq, change.Id, change.Rev)  // prints change 'seq', 'id' and 'rev'

        // Doc body
        str, _ := json.MarshalIndent(change.Doc, "", "  ")
        fmt.Printf("%s\n", str)
    } else {
        break
    }
}
```

### `db.NewFollower(seq)`

Creates a new changes feed follower that runs in continuous mode, emitting
events from the changes feed on a channel. Its aims is to stay running until told to
terminate with `changes.Close()`

```go
// Only generate a Seq ID every 100 changes
follower := db.NewFollower(100)
changes, err := follower.Follow()
if err != nil {
    fmt.Println(err)
    return
}

for {
    changeEvent := <-changes

    switch changeEvent.EventType {
    case cloudant.ChangesHeartbeat:
        fmt.Println("tick")
    case cloudant.ChangesError:
        fmt.Println(changeEvent.Err)
    case cloudant.ChangesTerminated:
        fmt.Println("terminated; resuming from last known sequence id")
        changes, err = follower.Follow()
        if err != nil {
            fmt.Println("resumption error ", err)
            return
        }
    case cloudant.ChangesInsert:
        fmt.Printf("INSERT %s\n", changeEvent.Meta.ID)
    case cloudant.ChangesDelete:
        fmt.Printf("DELETE %s\n", changeEvent.Meta.ID)
    default:
        fmt.Printf("UPDATE %s\n", changeEvent.Meta.ID)
    }
}
```


## Document functions

### `db.Destroy(docID, rev)`

Removes a document from database

```go
err := db.Destroy("my_doc_id", "2-xxxxxxx")
```

### `db.Get(docID, docQuery, doc)`

Gets a document from Cloudant whose _id is docID and unmarshals it into doc struct
```go
type Doc struct {
    Id     string    `json:"_id"`
    Rev    string    `json:"_rev"`
    Foo    string    `json:"foo"`
}

doc = new(Doc)
err = db.Get("my_doc_id", cloudant.NewDocQuery(), doc)

fmt.Println(doc.Foo)  // prints 'foo' key
```

### `db.Insert(doc)`

Inserts `myDoc` in the database

```go
myDoc := &Doc{
    ID:     "my_doc_id",
    Foo:    "bar",
}

newRev, err := db.Insert(myDoc)

fmt.Println(newRev)  // prints '_rev' of new document revision
```

### `db.InsertEscaped(doc)`

Inserts `myDoc` in the database and escapes HTML in strings

```go
newRev, err := db.InsertEscaped(myDoc)
```

### `db.InsertRaw(json)`

Inserts raw JSON ([]byte) in the database

```go
json := []byte(`{
		"_id": "_design/test_design_doc",
		"language": "javascript",
		"views": {
			"start_with_one": {
				"map": "function (doc) { if (doc._id.indexOf('-01') > 0) emit(doc._id, doc.foo) }"
			}
		}
	  }`)

newRev, err := db.InsertRaw(json)
```

### `db.Bulk(batchSize, batchMaxBytes, flushSecs)`
Bulk operations(update/delete/insert) on the database's `/_bulk_docs` endpoint

```go
myDoc1 := Doc{
    ID:     "doc1",
    Foo:    "bar",
}

myDoc2 := Doc{
    ID:     "doc2",
    Foo:    "bar",
}

myDoc3 := Doc{
    ID:     "doc3",
    Foo:    "bar",
}

uploader := db.Bulk(50, 1048576, 60) // new uploader using batch size 50, max batch size 1MB, flushing documents to server every 60 seconds

// Note: workers only flush their document batch to the server:
//  1)  periodically (set to -1 to disable).
//  2)  when the maximum number of documents per batch is reached.
//  3)  when the maximum batch size (in bytes) is reached (set to -1 to disable).
//  4)  if a document is uploaded using `.UploadNow(doc)`.
//  5)  if a client calls `.Flush()` or `.Stop()`.

uploader.FireAndForget(myDoc1)

upload.Flush() // blocks until all received documents have been uploaded

r2 := uploader.UploadNow(myDoc2) // uploaded as soon as it's received by a worker

r2.Wait()
if r2.Error != nil {
    fmt.Println(r2.Response.Id, r2.Response.Rev) // prints new document '_id' and 'rev'
}

r3 := uploader.Upload(myDoc3) // queues until the worker creates a full batch of 50 documents

upload.AsyncFlush() // asynchronously uploads all received documents

upload.Stop() // blocks until all documents have been uploaded and workers have stopped

r3.Wait()
if r3.Error != nil {
    fmt.Println(r3.Response.Id, r3.Response.Rev) // prints new document '_id' and 'rev'
}
```

## Views functions 

### `db.List(viewQuery)`

List all the docs in the database (`_all_docs` view)

```go
rows, err := db.List(cloudant.NewViewQuery())

// OR include some query options...
//
// q := cloudant.NewViewQuery().
//        Limit(123).
//        StartKey("foo1").
//        EndKey("foo2")
//
// rows, err := db.List(q)

for {
    row, more := <-rows
    if more {
        fmt.Println(row.ID, row.Value.Rev)  // prints document 'id' and 'rev'
    } else {
        break
    }
}
```

### `db.View(designName, viewName, viewQuery, view)`

Calls a view of the specified designName and decodes the result using provided interface

```go
type MyRow struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Doc   MyDoc  `json:"doc"`
}
type MyDoc struct {
	ID  string `json:"_id,omitempty"`
	Rev string `json:"_rev,omitempty"`
	Foo string `json:"foo" binding:"required"`
}


myView := new(DocsView)
q := cloudant.NewViewQuery().
    Descending().
    Key("foo1").
    Limit(1).
    IncludeDocs()

rows, err := db.View("my_design_name", "my_view_name", q, myView)

for {
    row, more := <-rows
    if more {
        r := new(MyRow)
        err = json.Unmarshal(row.([]byte), r)
        if err == nil {
            fmt.Println(r.Doc.ID, r.Doc.Rev, r.Doc.Foo)  // prints document 'id', 'rev', and 'foo' value
        }
    } else {
        break
    }
}
```

### `db.ViewRaw(designName, viewName, viewQuery)`

Calls a view of the specified designName and returns raw []byte response. This allows querying views with arbitrary output such as when using reduce.

```go
import "github.com/buger/jsonparser"

response, err := db.ViewRaw("my_design_name", "my_view_name", cloudant.NewViewQuery())

if err != nil {
    value, decodeErr := jsonparser.Get(response, "total_rows")
}
```