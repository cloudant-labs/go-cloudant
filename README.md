# go-cloudant
A Cloudant library for Golang.

[![Build Status](https://travis-ci.org/smithsz/go-cloudant.svg?branch=master)](https://travis-ci.org/smithsz/go-cloudant)

_The API is not fully baked at this time and may change._

## Description
A [Cloudant](https://cloudant.com/) library for Golang.

## Installation
```bash
go get github.com/smithsz/go-cloudant
```

## Supported Features
- Session authentication
- Keep-Alive & Connection Pooling
- Hard limit on request concurrency
- Stream `/_all_docs` & `/_changes`

## Getting Started

### `Get` a document:
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

### `Set` a document:
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

### Get `/_all_docs`:
```go
// create a Cloudant client (max. request concurrency 5)
client, err := cloudant.CreateClient("user123", "pa55w0rd01", "https://user123.cloudant.com", 5)
db, err := client.GetOrCreate("my_database")

allDocs, err := db.AllDocs()

for{
    doc, more := <-allDocs
	if more {
	    fmt.Println(doc.Id, doc.Value.Rev)  // prints document 'id' and 'rev'
	} else {
	    break
	}
}
```

### Get `/_changes`:
```go
// create a Cloudant client (max. request concurrency 5)
client, err := cloudant.CreateClient("user123", "pa55w0rd01", "https://user123.cloudant.com", 5)
db, err := client.GetOrCreate("my_database")

changes, err := db.Changes()

for{
    change, more := <-Changes
	if more {
	    fmt.Println(doc.Seq, doc.Id, doc.Rev)  // prints change 'seq', 'id' and 'rev'
	} else {
	    break
	}
}
```
