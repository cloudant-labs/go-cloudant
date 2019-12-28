# syntax=docker/dockerfile:1-experimental
# 1. Build as /build/app
FROM golang:1.13-stretch as builder
WORKDIR /tmp/app

# install
RUN apt-get update && \
    apt-get install git

# cache phase
COPY /go.mod .
COPY /go.sum .
RUN go mod download

# build phase
COPY / .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o app .

# 2. Run
FROM golang:1.13-stretch
ENV PORT 3000
ENV GIN_MODE release
EXPOSE $PORT
COPY --from=builder /tmp/app /cmd/app/app
WORKDIR /cmd/app
CMD ["./app"]
