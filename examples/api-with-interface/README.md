# go-cloudant API example using `cloudanti`

API example using go-cloudant library and GIN that uses `cloudanti` interface wrapper

Shows mocking example using `cloudanti` mock implementation

## Development

1. Use Go 1.13+
2. Set up your `.env `file (copy and modify `.env.example`)

### (Optional) Direct build with hot reload
Run `./run dev`

### (Optional) Dockerized build with hot reload
1. Add your key to SSH agent (required to download common-go-helper) `ssh-add -K ~/.ssh/id_rsa` (one time)
2. Run `./run dev-docker`

