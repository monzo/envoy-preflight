package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

var (
	goodServer           *httptest.Server
	goodEventuallyServer *httptest.Server
	badServer            *httptest.Server
	testsInit            bool  = false
	envoyDelayTimestamp  int64 = 0
	envoyDelayMax        int64 = 15
)

// Sets up default env variables and mock http servers
// Can be called multiple times, but will only init once per test session
func initTestingEnv() {
	if testsInit {
		return
	}
	os.Setenv("SCUTTLE_LOGGING", "true")
	os.Setenv("START_WITHOUT_ENVOY", "false") // If your tests never finish, this failed

	// Always 200 and live envoy state
	goodServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"state\": \"LIVE\"}")) // Envoy live response
	}))

	// 503 for 5 requests, then 200 + live envoy state
	goodEventuallyServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeSinceStarted := time.Now().Unix() - envoyDelayTimestamp
		if timeSinceStarted < envoyDelayMax {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("{\"state\": \"LIVE\"}")) // Envoy live response
	}))

	// Always 503
	badServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	testsInit = true
}

// Tests START_WITHOUT_ENVOY works with failing envoy mock server
func TestBlockingDisabled(t *testing.T) {
	os.Setenv("START_WITHOUT_ENVOY", "true")
	block("")
	// If your tests hang and never finish, this test "failed"
	// Also try go test -timeout <seconds>s
}

// Tests block function with working envoy mock server
func TestBlockingEnabled(t *testing.T) {
	initTestingEnv()
	os.Setenv("START_WITHOUT_ENVOY", "false")
	block(goodServer.URL)
}

// Tests block function with envoy mock server that fails for 15 seconds, then works
func TestSlowEnvoy(t *testing.T) {
	initTestingEnv()
	os.Setenv("START_WITHOUT_ENVOY", "false")
	envoyDelayTimestamp = time.Now().Unix()
	block(goodEventuallyServer.URL)
}
