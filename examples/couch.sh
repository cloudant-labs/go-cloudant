#!/bin/bash
name=couchdb
export COUCH_USER="mrblobby"
export COUCH_PASS="blobbypassword"
export COUCH_HOST_URL="http://127.0.0.1:5984"

#Check docker is running
docker_state=$(docker info >/dev/null 2>&1)
if [[ $? -ne 0 ]]; then
    echo "Docker does not seem to be running"
    exit 1
fi

#Check container exists
container_exists=$(docker inspect $name)
if [[ $container_exists == "[]" ]]; then
    echo "Intalling $name ..."
    # Start CouchDB docker
    docker run -d -p 5984:5984 --rm --name couchdb couchdb:1.6
    curl -X PUT $COUCH_HOST_URL/_config/admins/$COUCH_USER -d '"'$COUCH_PASS'"'
    exit 1
fi 

#Check container is running
container_state=$(docker inspect -f '{{.State.Running}}' $name)
if [[ $container_state == "true" ]]; then
    echo "$name is already running"
    docker ps
    exit 1
fi

#Run
echo "Starting $name ..."
docker start $name
docker ps
