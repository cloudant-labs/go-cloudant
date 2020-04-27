# Contributing

### Running the Tests

To run the tests, you need a Cloudant (or CouchDB) database to talk to. The tests
expect the following environment variables to be available:

```sh
COUCH_USER
COUCH_PASS
COUCH_HOST_URL # optional
```

If the last one isn't set, the host url is assumed to be `https://$COUCH_USER.cloudant.com`.

If you want to run against a local CouchDB in Docker, try [this set-up script](examples/couch.sh) or:

```sh
docker run -d -p 5984:5984 --rm --name couchdb couchdb:1.6
curl -X PUT 'http://127.0.0.1:5984/_config/admins/mrblobby' -d '"blobbypassword"'
export COUCH_USER="mrblobby"
export COUCH_PASS="blobbypassword"
export COUCH_HOST_URL="http://127.0.0.1:5984"
go test
```

Note -- this library does not allow for unauthenticated connections, so you can't
run against a CouchDB node in `admin party` mode. This is a good thing. This also means you can't use couchdb service in Travis, see [working Travis configuration](.travis.yml).

## Issues

Please [read these guidelines](http://ibm.biz/cdt-issue-guide) before opening an issue.
If you still need to open an issue then we ask that you complete the template as
fully as possible.

## Pull requests

We welcome pull requests, but ask contributors to keep in mind the following:

* Only PRs with the template completed will be accepted
* We will not accept PRs for user specific functionality

### Developer Certificate of Origin

In order for us to accept pull-requests, the contributor must sign-off a
[Developer Certificate of Origin (DCO)](DCO1.1.txt). This clarifies the
intellectual property license granted with any contribution. It is for your
protection as a Contributor as well as the protection of IBM and its customers;
it does not change your rights to use your own Contributions for any other purpose.

Please read the agreement and acknowledge it by ticking the appropriate box in the PR
 text, for example:

- [x] Tick to sign-off your agreement to the Developer Certificate of Origin (DCO) 1.1

<!-- Append library specific information here

## General information

## Requirements

## Building

## Testing

 -->
