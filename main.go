package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/cenk/backoff"
	"github.com/monzo/typhon"
)

var loggingEnabled = false

type ServerInfo struct {
	State string `json:"state"`
}

func log(message string) {
	fmt.Println("scuttle: " + message)
}

func main() {
	// Check if logging is enabled
	loggingEnabled = (os.Getenv("SCUTTLE_LOGGING") == "true")
	if loggingEnabled {
		log("Logging is now enabled")
	}

	// Should be in format `http://127.0.0.1:9010`
	host, ok := os.LookupEnv("ENVOY_ADMIN_API")
	log(fmt.Sprintf("ENVOY_ADMIN_API: %s", host))

	startWithoutEnvoy := os.Getenv("START_WITHOUT_ENVOY")
	log(fmt.Sprintf("START_WITHOUT_ENVOY: %s", startWithoutEnvoy))

	if ok && startWithoutEnvoy != "true" {
		log("Blocking host until envoy starts")
		block(host)
	}

	if len(os.Args) < 2 {
		log("No arguments received, exiting")
		return
	}

	// Find the executable the user wants to run
	binary, err := exec.LookPath(os.Args[1])
	if err != nil {
		panic(err)
	}

	var proc *os.Process

	// Pass signals to the child process
	go func() {
		stop := make(chan os.Signal, 2)
		signal.Notify(stop)
		for sig := range stop {
			if proc != nil {
				proc.Signal(sig)
			} else {
				// Signal received before the process even started. Let's just exit.
				log("Received exit signal, exiting")
				os.Exit(1)
			}
		}
	}()

	// Start process passed in by user
	proc, err = os.StartProcess(binary, os.Args[1:], &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		panic(err)
	}

	state, err := proc.Wait()
	if err != nil {
		panic(err)
	}

	exitCode := state.ExitCode()

	switch {
	case !ok:
		// We don't have an ENVOY_ADMIN_API env var, do nothing
		log("No ENVOY_ADMIN_API, doing nothing")
	case !strings.Contains(host, "127.0.0.1") && !strings.Contains(host, "localhost"):
		// Envoy is not local; do nothing
		log("ENVOY_ADMIN_API is not localhost or 127.0.0.1, doing nothing")
	case os.Getenv("NEVER_KILL_ENVOY") == "true":
		// We're configured never to kill envoy, do nothing
		log("NEVER_KILL_ENVOY is true, doing nothing")
	default:
		// Either we had a clean exit, or we are configured to kill istio anyway
		cmd := exec.Command("sh", "-c", "pkill -SIGINT pilot-agent")
		_, err := cmd.Output()
		if err == nil {
			log("Process pilot-agent successfully stopped")
		} else {
			errorMessage := err.Error()
			log("pilot-agent could not be stopped, err: " + errorMessage)
		}
	}

	os.Exit(exitCode)
}

func block(host string) {
	if os.Getenv("START_WITHOUT_ENVOY") == "true" {
		return
	}

	url := fmt.Sprintf("%s/server_info", host)

	b := backoff.NewExponentialBackOff()
	// We wait forever for envoy to start. In practice k8s will kill the pod if we take too long.
	b.MaxElapsedTime = 0

	_ = backoff.Retry(func() error {
		rsp := typhon.NewRequest(context.Background(), "GET", url, nil).Send().Response()

		info := &ServerInfo{}

		err := rsp.Decode(info)
		if err != nil {
			return err
		}

		if info.State != "LIVE" {
			return errors.New("not live yet")
		}

		return nil
	}, b)
}
