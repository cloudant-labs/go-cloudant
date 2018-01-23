package cloudant

import (
	"os"
	"testing"
)

func TestInvalidLogin(t *testing.T) {
	username := os.Getenv("COUCH_USER")
	password := "wR0ng_pa$$w0rd"
	host := os.Getenv("COUCH_HOST_URL")

	if host == "" {
		host = "https://" + username + ".cloudant.com"
	}
	if username == "" {
		t.Fatalf("expected env var COUCH_USER to be set")
	}

	_, err := CreateClient(username, password, host, 5)

	if err == nil {
		t.Errorf("missing error from invalid login attempt")
	}
	if err.Error() != "failed to create session, status 401" {
		t.Errorf("unexpected error message: %s", err)
	}
}
