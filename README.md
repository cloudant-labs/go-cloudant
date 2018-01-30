# go-cloudant

A Cloudant library for Golang.

[![Build Status](https://travis-ci.org/cloudant-labs/go-cloudant.svg?branch=master)](https://travis-ci.org/cloudant-labs/go-cloudant)

__This library is not complete, may change in incompatible ways in future versions, and comes with no support whatsoever.__

## Description

A [Cloudant](https://cloudant.com/) library for Golang.

## Installation

```bash
go get github.com/cloudant-labs/go-cloudant
```

## Current Features

- Session authentication
- Keep-Alive & Connection Pooling
- Configurable request retrying
- Hard limit on request concurrency
- Stream `/_all_docs` & `/_changes`
- Manage `/_bulk_docs` uploads

## Getting Started

### Running the Tests

To run the tests, you need a Cloudant (or CouchDB) database to talk to. The tests
expect the following environment variables to be available:

```sh
COUCH_USER
COUCH_PASS
COUCH_HOST_URL # optional
```

If the last one isn't set, the host url is assumed to be `https://$COUCH_USER.cloudant.com`.

If you want to run against a local CouchDB in Docker, try:

```sh
docker run -d -p 5984:5984 --rm --name couchdb couchdb:1.6
curl -XPUT 'http://127.0.0.1:5984/_config/admins/mrblobby' -d '"blobbypassword"'
export COUCH_USER="mrblobby"
export COUCH_PASS="blobbypassword"
export COUCH_HOST_URL="http://127.0.0.1:5984"
go test
```

Note -- this library does not allow for unauthenticated connections, so you can't
run against a CouchDB node in `admin party` mode. This is a good thing.

### Creating a Cloudant client

```go
// create a Cloudant client (max. request concurrency 5) with default retry configuration:
//   - maximum retries per request:     3
//   - random retry delay minimum:      5  seconds
//   - random retry delay maximum:      30 seconds
client1, err1 := cloudant.CreateClient("user123", "pa55w0rd01", "https://user123.cloudant.com", 5)

// create a Cloudant client (max. request concurrency 20) with _custom_ retry configuration:
//   - maximum retries per request:     5
//   - random retry delay minimum:      10  seconds
//   - random retry delay maximum:      60 seconds
client2, err2 := cloudant.CreateClientWithRetry("user123", "pa55w0rd01", "https://user123.cloudant.com", 20, 5, 10, 60)
```

### `Get` a document

```go
// create a Cloudant client (max. request concurrency 5)
client, err := cloudant.CreateClient("user123", "pa55w0rd01", "https://user123.cloudant.com", 5)
db, err := client.GetOrCreate("my_database")

type Doc struct {
    Id     string    `json:"_id"`
    Rev    string    `json:"_rev"`
    Foo    string    `json:"foo"`
}

doc = new(Doc)
err = db.Get("my_doc", doc)

fmt.Println(doc.Foo)  // prints 'foo' key
```

### `Set` a document

```go
// create a Cloudant client (max. request concurrency 5)
client, err := cloudant.CreateClient("user123", "pa55w0rd01", "https://user123.cloudant.com", 5)
db, err := client.GetOrCreate("my_database")

myDoc := &Doc{
        Id:     "my_doc_id",
        Rev:    "2-xxxxxxx",
        Foo:    "bar",
}

newRev, err := db.Set(myDoc)

fmt.Println(newRev)  // prints '_rev' of new document revision
```

### `Delete` a document

```go
// create a Cloudant client (max. request concurrency 5)
client, err := cloudant.CreateClient("user123", "pa55w0rd01", "https://user123.cloudant.com", 5)
db, err := client.GetOrCreate("my_database")

err := db.Delete("my_doc_id", "2-xxxxxxx")
```

### Using `/_bulk_docs`

```go
// create a Cloudant client (max. request concurrency 5)
client, err := cloudant.CreateClient("user123", "pa55w0rd01", "https://user123.cloudant.com", 5)
db, err := client.GetOrCreate("my_database")

myDoc1 := Doc{
        Id:     "doc1",
        Rev:    "1-xxxxxxx",
        Foo:    "bar",
}

myDoc2 := Doc{
        Id:     "doc2",
        Rev:    "2-xxxxxxx",
        Foo:    "bar",
}

myDoc3 := Doc{
        Id:     "doc3",
        Rev:    "3-xxxxxxx",
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

### Using `/_all_docs`

```go
// create a Cloudant client (max. request concurrency 5)
client, err := cloudant.CreateClient("user123", "pa55w0rd01", "https://user123.cloudant.com", 5)
db, err := client.GetOrCreate("my_database")

rows, err := db.All(&allDocsQuery{})

// OR include some query options...
//
// q := cloudant.NewAllDocsQuery().
//        Limit(123).
//        StartKey("foo1").
//        EndKey("foo2").
//        Build()
//
//    rows, err := db.All(q)

for{
    row, more := <-rows
    if more {
        fmt.Println(row.ID, row.Value.Rev)  // prints document 'id' and 'rev'
    } else {
        break
    }
}
```

### Using `/_changes`

```go
// create a Cloudant client (max. request concurrency 5)
client, err := cloudant.CreateClient("user123", "pa55w0rd01", "https://user123.cloudant.com", 5)
db, err := client.GetOrCreate("my_database")

query := cloudant.NewChangesQuery().IncludeDocs().Build()

changes, err := db.Changes(query)

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

### Using `Follower`

`Follower` is a robust changes feed follower that runs in continuous mode, emitting
events from the changes feed on a channel. Its aims is to stay running until told to
terminate.

```go
client, err := cloudant.CreateClient(...)

db, err := client.Get(DATABASE)
if err != nil {
    fmt.Printf("error\n")
    return
}

// Only generate a Seq ID every 100 changes
follower := cloudant.NewFollower(db, 100)
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
