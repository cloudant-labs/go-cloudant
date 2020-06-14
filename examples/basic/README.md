# basic - basic example program

### Prequisites:
* Access to a Cloudant instance. This example has a .env file where you can put your Cloudant instance details.
If you do not have access to a Cloudant instance, you can install [Docker](https://www.docker.com/community-edition) and use the [Couch DB Docker Container](https://hub.docker.com/_/couchdb).
* A configured [Go environment](https://golang.org/doc/install).

### Before running the basic example program (main.go):
This program uses two packages so you will need to get them first so that they are available on your local machine and in the correct directory.
From your go src directory issue the following commands:

```
go get github.com/joho/godotenv
```

```
go get github.com/cloudant-labs/go-cloudant
```

Download the basic folder with contents to your go src directory.

### Running the basic example program (main.go):
Ensure that you are in the basic folder on your local machine, you have done the previous steps in this README file, your cloudant-developer Docker Container is running (if applicable), then issue the following command:

```
go run main.go
```

You should see output similar to the following output:

```
2018/01/19 14:25:12 Request (attempt: 0) POST http://localhost:5984/_session
2018/01/19 14:25:12 Connected to Cloudant Successfully
2018/01/19 14:25:12 Request (attempt: 0) HEAD http://localhost:5984
2018/01/19 14:25:12 Request (attempt: 0) PUT http://localhost:5984/items
2018/01/19 14:25:12 Request (attempt: 0) POST http://localhost:5984/items
2018/01/19 14:25:12 &{2ee2228864b504d52ad38f445f006596 1-9fdc2238b02e29dd1fda502b7ff07157}
```


