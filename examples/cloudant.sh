#!/bin/bash
name=cloudant-developer

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
    # Start cloudant developer docker https://hub.docker.com/r/ibmcom/cloudant-developer/
    docker run \
        --detach \
        --volume cloudant:/srv \
        --name cloudant-developer \
        --publish 8080:80 \
        --hostname cloudant.dev \
        ibmcom/$name
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
