# go-cloudant
A Cloudant library for golang.

## Description
A [Cloudant](https://cloudant.com/) library for golang.

## Installation
```bash
go get github.com/smithsz/go-cloudant
```

## Supported Features
- Session authentication
- Keep-Alive & Connection Pooling
- Hard limit on request concurrency
- Stream `/_all_docs`

## Documentation
_TODO_

## Getting Started

### Get a document:
```go
// create a Cloudant client (max. request concurrency 5)
client, err := cloudant.CreateClient("username", "xxxxxxxx", "https://username.cloudant.com", 5)
db, err := client.GetOrCreateDatabase("my_database")

doc = Doc{
    Id  string  `json:"_id"`
    Rev string  `json:"_rev"`
    Foo string  `json:"foo"`
}

err = db.GetDocument("my_doc", doc)

fmt.Println(doc.Foo)  // prints 'foo' key
```

### Get `/_all_docs`:
```go
// create a Cloudant client (max. request concurrency 5)
client, err := cloudant.CreateClient("username", "xxxxxxxx", "https://username.cloudant.com", 5)
db, err := client.GetOrCreateDatabase("my_database")

allDocs, err := db.getAllDocs()

for{
    doc, more := <-allDocs
	if more {
	    fmt.Println(doc.Id, doc.Value.Rev)  // print '_id' and '_rev'
	} else {
	    break
	}
}
```
