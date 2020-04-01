package main

import (
	"fmt"
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
	genericQuitServer    *httptest.Server
	testsInit            bool  = false
	envoyDelayTimestamp  int64 = 0
	envoyDelayMax        int64 = 15
)

// Sets up minimum env variables and mock http servers
// Can be called multiple times, but will only init once per test session
func initTestingEnv() {
	// Always update env variables for new test
	os.Setenv("SCUTTLE_LOGGING", "true")
	config = getConfig()

	// Do not restart http servers for each test
	if testsInit {
		return
	}

	fmt.Println("Initing test HTTP servers")

	// Always 200 and live envoy state
	goodServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"state\": \"LIVE\"}")) // Envoy live response
	}))

	// 503 for 5 requests, then 200 + live envoy state
	goodEventuallyServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeSinceStarted := time.Now().Unix() - envoyDelayTimestamp
		if timeSinceStarted < envoyDelayMax {
			fmt.Println("Status Unavailable")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("{\"state\": \"LIVE\"}")) // Envoy live response
	}))

	// Always 503
	badServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Status Unavailable")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	genericQuitServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Status Ok")
		w.WriteHeader(http.StatusOK)
	}))

	testsInit = true
}

// Inits the test environment and starts the blocking
// Set any env variables for your specific tests prior to calling this
// Pass in a negative integer to block but skip kill
func initAndRun(exitCode int) {
	initTestingEnv()
	block()
	if exitCode >= 0 {
		kill(exitCode)
	}
}

// Tests START_WITHOUT_ENVOY works with failing envoy mock server
func TestBlockingDisabled(t *testing.T) {
	fmt.Println("Starting TestBlockingDisabled")
	os.Setenv("START_WITHOUT_ENVOY", "true")
	initAndRun(-1)
	// If your tests hang and never finish, this test "failed"
	// Also try go test -timeout <seconds>s
}

// Tests block function with working envoy mock server
func TestBlockingEnabled(t *testing.T) {
	fmt.Println("Starting TestBlockingEnabled")
	os.Setenv("START_WITHOUT_ENVOY", "false")
	os.Setenv("ENVOY_ADMIN_API", goodServer.URL)
	initAndRun(-1)
}

// Tests block function with envoy mock server that fails for 15 seconds, then works
func TestSlowEnvoy(t *testing.T) {
	fmt.Println("Starting TestSlowEnvoy")
	os.Setenv("START_WITHOUT_ENVOY", "false")
	os.Setenv("ENVOY_ADMIN_API", goodEventuallyServer.URL)
	envoyDelayTimestamp = time.Now().Unix()
	initAndRun(-1)
}

// Tests generic quit endpoints are sent
func TestGenericQuitEndpoints(t *testing.T) {
	fmt.Println("Starting TestGenericQuitEndpoints")
	// Valid URLs dont matter, just need something that will generate an HTTP response
	// 127.0.0.1:1111/idontexist is to verify we don't panic if a nonexistent URL is given
	// notaurl^^ is to verify a malformatted URL does not result in panic
	os.Setenv("GENERIC_QUIT_ENDPOINTS", "https://google.com/, https://github.com/, 127.0.0.1:1111/idontexist, notaurl^^ ")
	initTestingEnv()
	killGenericEndpoints()
}

// Tests scuttle does not fail when the /quitquitquit endpoint does not return a response
func TestNoQuitQuitQuitResponse(t *testing.T) {
	fmt.Println("Starting TestNoQuitQuitQuitResponse")
	os.Setenv("START_WITHOUT_ENVOY", "false")
	os.Setenv("ISTIO_QUIT_API", "127.0.0.1:1111/idontexist")
	initTestingEnv()
	killIstioWithAPI()
}

// Tests scuttle does not fail when the /quitquitquit endpoint is not a valid URL
func TestNoQuitQuitQuitMalformattedUrl(t *testing.T) {
	fmt.Println("Starting TestNoQuitQuitQuitMalformattedUrl")
	os.Setenv("START_WITHOUT_ENVOY", "false")
	os.Setenv("ISTIO_QUIT_API", "notaurl^^")
	initTestingEnv()
	killIstioWithAPI()
}
