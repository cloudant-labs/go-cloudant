package cloudant

import (
	"net/http"
	"testing"
)

func TestCouchClient_LogIn(t *testing.T) {
	setupClient()

	if len(capturedJobs) != 1 {
		t.Error("unexpected request sent to server")
	}

	job := capturedJobs[0]
	if "POST" != job.request.Method {
		t.Errorf("unexpected request method %s", job.request.Method)
	}

	if testUsername != job.request.FormValue("name") {
		t.Errorf("unexpected name value %s", job.request.FormValue("name"))
	}

	if testPassword != job.request.FormValue("password") {
		t.Errorf("unexpected password value %s", job.request.FormValue("password"))
	}

	if "https://"+testUsername+".cloudant.com/_session" != job.request.URL.String() {
		t.Errorf("unexpected request URL %s", job.request.URL.String())
	}
}

func TestCouchClient_LogOut(t *testing.T) {
	client := setupClient()

	setupMocks([]*http.Response{mock200})

	client.LogOut()

	if len(capturedJobs) != 1 {
		t.Error("unexpected request sent to server")
	}

	job := capturedJobs[0]
	if "DELETE" != job.request.Method {
		t.Errorf("unexpected request method %s", job.request.Method)
	}

	if "https://"+testUsername+".cloudant.com/_session" != job.request.URL.String() {
		t.Errorf("unexpected request URL %s", job.request.URL.String())
	}
}

func TestCouchClient_Ping(t *testing.T) {
	client := setupClient()

	setupMocks([]*http.Response{mock200})

	client.Ping()

	if len(capturedJobs) != 1 {
		t.Error("unexpected request sent to server")
	}

	job := capturedJobs[0]
	if "HEAD" != job.request.Method {
		t.Errorf("unexpected request method %s", job.request.Method)
	}

	if "https://"+testUsername+".cloudant.com" != job.request.URL.String() {
		t.Errorf("unexpected request URL %s", job.request.URL.String())
	}
}

func TestCouchClient_GetOrCreate(t *testing.T) {
	client := setupClient()

	setupMocks([]*http.Response{mock200})

	client.GetOrCreate(testDatabaseName)

	if len(capturedJobs) != 1 {
		t.Error("unexpected request sent to server")
	}

	job := capturedJobs[0]
	if "PUT" != job.request.Method {
		t.Errorf("unexpected request method %s", job.request.Method)
	}

	if "https://"+testUsername+".cloudant.com/"+testDatabaseName != job.request.URL.String() {
		t.Errorf("unexpected request URL %s", job.request.URL.String())
	}
}
