# cloudanti: go-cloudant interface

`cloudanti` provides a convenience interface wrapper and mock functions for go-cloudant. The interface wraps and mocks all main library functions except for database's `Bulk`, `Changes` and `NewFollower`.

The package can be used directly (see [basic](../examples/basic-with-interface) or [api](../examples/api-with-interface) examples) or copied and customized as required. 

## Getting started

Use `cloudanti` instead of `cloudant` to set up the client.

Refer to the [full example](../examples/basic-with-interface) for a working implementation.

### main.go

```go
import "github.com/cloudant-labs/go-cloudant/interface"

// Create Cloudant Client
client, err := cloudanti.NewClient("user123", "pa55w0rd01", "https://user123.cloudant.com")
```

### main_test.go

```go
// Create Cloudant Mock Client
client, _ := cloudanti.NewMockClient(cloudanti.CloudantContent{})
```

## Mock content

Define mock content with mock docs and/or the list of document IDs to return in a view.

Views are tricky to mock, please be aware of the severe limitations:
- View string must match the library call exactly, including the order of any provided ViewQuery parameters
- Defined views are static, only the document contents will be updated
- Undefined views return all docs in the mock database

```go
content := cloudanti.CloudantContent{
    Databases: map[string]cloudanti.DatabaseContent{
        "my_db_name": {
            Docs: map[string][]byte{
                "my_doc_id1": []byte(`{"_id":"my_doc_id1","_rev":"34-23412341324","foo":"bar"}`),
                "my_doc_id2": []byte(`{"_id":"my_doc_id2","_rev":"1-12374912709","foo":"baz"}`),
            },
            Views: map[string][]string{
                "/_design/my_design_name/_view/my_view_name?descending=true": []string("my_doc_id2", "my_doc_id1"),
            },
        },
    },
}

client, _ := cloudanti.NewMockClient(content)
```

Refer to the [api example](../examples/api-with-interface/controller_test.go) for a working implementation.
